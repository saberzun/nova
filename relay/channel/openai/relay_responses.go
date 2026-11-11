package openai

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type responsesUsageObserver struct {
	mu                    sync.Mutex
	frozen                bool
	cutoff                uint64
	uncertain             bool
	uncertaintyReason     string
	authoritativeSeen     bool
	authoritativeUsage    *dto.Usage
	authoritativeValid    bool
	authoritativeRejected string
	legacyUsage           *dto.Usage
	normalTerminalSeen    bool
	terminalFailureSeen   bool
	seenEventIDs          map[string]struct{}
	textByItem            map[string]*strings.Builder
	reasoningByItem       map[string]*strings.Builder
	argumentsByItem       map[string]*strings.Builder
	textParts             []string
	reasoningParts        []string
	argumentParts         []string
}

func newResponsesUsageObserver() *responsesUsageObserver {
	return &responsesUsageObserver{
		seenEventIDs:    make(map[string]struct{}),
		textByItem:      make(map[string]*strings.Builder),
		reasoningByItem: make(map[string]*strings.Builder),
		argumentsByItem: make(map[string]*strings.Builder),
	}
}

func (o *responsesUsageObserver) Observe(data string, sequence uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frozen && sequence > o.cutoff {
		return
	}
	eventID := gjson.Get(data, "event_id").String()
	if eventID == "" {
		if sequenceNumber := gjson.Get(data, "sequence_number"); sequenceNumber.Exists() {
			eventID = "sequence:" + sequenceNumber.Raw
		}
	}
	if eventID != "" {
		if _, exists := o.seenEventIDs[eventID]; exists {
			return
		}
		o.seenEventIDs[eventID] = struct{}{}
	}
	var event dto.ResponsesStreamResponse
	if err := common.UnmarshalJsonStr(data, &event); err != nil {
		o.uncertain = true
		o.uncertaintyReason = "event_parse_failed"
		return
	}
	key := responseEventKey(event)
	switch event.Type {
	case "response.completed", "response.done":
		o.normalTerminalSeen = true
		if event.Response != nil && event.Response.Usage != nil {
			o.authoritativeSeen = true
			o.legacyUsage = legacyResponsesUsage(event.Response.Usage)
			usage, valid, reason := normalizeResponsesUsage(data, event.Response.Usage)
			if valid {
				o.authoritativeUsage = usage
				o.authoritativeValid = true
			} else {
				o.authoritativeRejected = reason
			}
		}
	case "response.failed", "response.incomplete", "response.cancelled":
		o.terminalFailureSeen = true
	case "response.output_text.delta":
		o.textParts = appendResponsePart(o.textParts, event.Delta)
		appendResponseItem(o.textByItem, key, event.Delta)
	case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
		value := event.Delta
		if value == "" && event.Part != nil {
			value = event.Part.Text
		}
		o.reasoningParts = appendResponsePart(o.reasoningParts, value)
		appendResponseItem(o.reasoningByItem, key, value)
	case "response.function_call_arguments.delta":
		o.argumentParts = appendResponsePart(o.argumentParts, event.Delta)
		appendResponseItem(o.argumentsByItem, key, event.Delta)
	case "response.refusal.delta":
		o.textParts = appendResponsePart(o.textParts, event.Delta)
		appendResponseItem(o.textByItem, "refusal:"+key, event.Delta)
	default:
		if gjson.Get(data, "response.usage").Exists() || gjson.Get(data, "usage").Exists() {
			o.uncertain = true
			o.uncertaintyReason = "unknown_usage_event"
		}
	}
}

func legacyResponsesUsage(source *dto.Usage) *dto.Usage {
	usage := &dto.Usage{}
	if source.InputTokens != 0 {
		usage.PromptTokens = source.InputTokens
	}
	if source.OutputTokens != 0 {
		usage.CompletionTokens = source.OutputTokens
	}
	if source.InputTokensDetails != nil {
		usage.PromptTokensDetails.CachedTokens = source.InputTokensDetails.CachedTokens
		usage.PromptTokensDetails.CacheWriteTokens = source.InputTokensDetails.CacheWriteTokens
		usage.PromptTokensDetails.CacheCreationInputTokens = source.InputTokensDetails.CacheCreationInputTokens
	}
	return usage
}

func (o *responsesUsageObserver) Freeze(cutoffSequence uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.frozen = true
	o.cutoff = cutoffSequence
}

func (o *responsesUsageObserver) finalize() (usage *dto.Usage, output, reasoning, arguments string, uncertain bool, normalTerminal, terminalFailure bool, authoritativeSeen, authoritativeValid bool, rejected string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.authoritativeUsage, strings.Join(o.textParts, ""), strings.Join(o.reasoningParts, ""), strings.Join(o.argumentParts, ""), o.uncertain, o.normalTerminalSeen, o.terminalFailureSeen, o.authoritativeSeen, o.authoritativeValid, o.authoritativeRejected
}

func appendResponsePart(parts []string, value string) []string {
	if value == "" {
		return parts
	}
	return append(parts, value)
}

func responseEventKey(event dto.ResponsesStreamResponse) string {
	if event.ItemID != "" {
		return fmt.Sprintf("%s:%d:%d", event.ItemID, responseIndex(event.ContentIndex), responseIndex(event.SummaryIndex))
	}
	return fmt.Sprintf("%d:%d:%d", responseIndex(event.OutputIndex), responseIndex(event.ContentIndex), responseIndex(event.SummaryIndex))
}

func responseIndex(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}

func appendResponseItem(items map[string]*strings.Builder, key, value string) {
	if value == "" {
		return
	}
	builder := items[key]
	if builder == nil {
		builder = &strings.Builder{}
		items[key] = builder
	}
	builder.WriteString(value)
}

func (o *responsesUsageObserver) outputTokenCount(model string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	count := func(items map[string]*strings.Builder) int {
		total := 0
		for _, builder := range items {
			total += service.CountTextToken(builder.String(), model)
		}
		return total
	}
	return count(o.textByItem) + count(o.reasoningByItem) + count(o.argumentsByItem)
}

func (o *responsesUsageObserver) legacyFallbackUsage(info *relaycommon.RelayInfo) *dto.Usage {
	o.mu.Lock()
	defer o.mu.Unlock()
	usage := &dto.Usage{}
	if o.legacyUsage != nil {
		*usage = *o.legacyUsage
	}
	if usage.CompletionTokens == 0 {
		output := strings.Join(o.textParts, "")
		if output != "" && info != nil {
			usage.CompletionTokens = service.CountTextToken(output, info.UpstreamModelName)
		}
	}
	if usage.PromptTokens == 0 && usage.CompletionTokens != 0 && info != nil {
		usage.PromptTokens = info.GetEstimatePromptTokens()
	}
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage
}

func normalizeResponsesUsage(data string, source *dto.Usage) (*dto.Usage, bool, string) {
	usagePath := gjson.Get(data, "response.usage")
	for _, field := range []string{"input_tokens", "output_tokens", "total_tokens"} {
		fieldValue := usagePath.Get(field)
		if !fieldValue.Exists() || fieldValue.Type != gjson.Number {
			return nil, false, "usage_required_field_missing_or_invalid"
		}
	}
	if source.InputTokens < 0 || source.OutputTokens < 0 || source.TotalTokens < 0 || source.TotalTokens != source.InputTokens+source.OutputTokens {
		return nil, false, "usage_counts_inconsistent"
	}
	if source.InputTokensDetails != nil {
		details := source.InputTokensDetails
		if details.CachedTokens < 0 || details.CacheWriteTokens < 0 || details.TextTokens < 0 || details.ImageTokens < 0 || details.AudioTokens < 0 {
			return nil, false, "usage_input_details_negative"
		}
		if details.CachedTokens > source.InputTokens || details.CacheWriteTokens > source.InputTokens || details.TextTokens > source.InputTokens || details.ImageTokens > source.InputTokens || details.AudioTokens > source.InputTokens || details.CachedTokens > source.InputTokens-details.CacheWriteTokens {
			return nil, false, "usage_input_details_exceed_total"
		}
	}
	outputDetails := source.CompletionTokenDetails
	if outputDetails.ReasoningTokens < 0 || outputDetails.TextTokens < 0 || outputDetails.ImageTokens < 0 || outputDetails.AudioTokens < 0 || outputDetails.ReasoningTokens > source.OutputTokens || outputDetails.TextTokens > source.OutputTokens || outputDetails.ImageTokens > source.OutputTokens || outputDetails.AudioTokens > source.OutputTokens {
		return nil, false, "usage_output_details_inconsistent"
	}
	copyUsage := *source
	if source.InputTokensDetails != nil {
		details := *source.InputTokensDetails
		copyUsage.InputTokensDetails = &details
		copyUsage.PromptTokensDetails = details
	}
	copyUsage.PromptTokens = source.InputTokens
	copyUsage.CompletionTokens = source.OutputTokens
	copyUsage.TotalTokens = source.TotalTokens
	return &copyUsage, true, ""
}

func responsesFallbackEligible(info *relaycommon.RelayInfo, observer *responsesUsageObserver) bool {
	if !constant.ClientGoneFallbackEnabled || info == nil || !info.IsStream || info.RelayMode != relayconstant.RelayModeResponses || strings.Split(info.RequestURLPath, "?")[0] != "/v1/responses" || info.GetFinalRequestRelayFormat() != types.RelayFormatOpenAIResponses || len(info.RequestConversionChain) != 1 || !constant.CountToken || info.GetEstimatePromptTokens() <= 0 || observer == nil {
		return false
	}
	if info.PriceData.UsePrice && info.TieredBillingSnapshot == nil {
		return false
	}
	if len(info.ParamOverrideAudit) > 0 || len(info.ParamOverride) > 0 || info.PassThroughRequestBody {
		return false
	}
	request, ok := info.Request.(*dto.OpenAIResponsesRequest)
	if !ok || request == nil || request.Stream == nil || !*request.Stream || request.PreviousResponseID != "" || len(request.Conversation) > 0 || len(request.Prompt) > 0 {
		return false
	}
	if !responsesToolsAreTokenOnly(request.Tools, request.ToolChoice) || len(request.Include) > 0 || len(request.ContextManagement) > 0 || len(request.Metadata) > 0 || len(request.Moderation) > 0 || len(request.PromptCacheKey) > 0 || len(request.PromptCacheOptions) > 0 || len(request.PromptCacheRetention) > 0 || request.ServiceTier != "" {
		return false
	}
	if responsesInputContainsUnsupportedData(request.Input) {
		return false
	}
	_, _, _, _, uncertain, normalTerminal, terminalFailure, _, valid, _ := observer.finalize()
	if uncertain || normalTerminal || terminalFailure || valid || info.StreamStatus == nil || info.StreamStatus.GetEndReason() != relaycommon.StreamEndReasonClientGone || !info.StreamStatus.InspectionComplete() {
		return false
	}
	_, frozen := info.StreamStatus.CutoffSequence()
	return frozen
}

func responsesToolsAreTokenOnly(tools, toolChoice []byte) bool {
	if len(tools) == 0 {
		return len(toolChoice) == 0
	}
	var toolList []map[string]any
	if err := common.Unmarshal(tools, &toolList); err != nil || len(toolList) == 0 {
		return false
	}
	for _, tool := range toolList {
		if toolType, _ := tool["type"].(string); toolType != "function" {
			return false
		}
	}
	if len(toolChoice) == 0 {
		return true
	}
	if common.GetJsonType(toolChoice) == "string" {
		var choice string
		if err := common.Unmarshal(toolChoice, &choice); err != nil {
			return false
		}
		return choice == "auto" || choice == "none" || choice == "required"
	}
	var choice map[string]any
	if err := common.Unmarshal(toolChoice, &choice); err != nil {
		return false
	}
	choiceType, _ := choice["type"].(string)
	return choiceType == "function"
}

func responsesInputContainsUnsupportedData(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return true
	}
	var visit func(any) bool
	visit = func(current any) bool {
		switch typed := current.(type) {
		case string:
			return false
		case []any:
			for _, item := range typed {
				if visit(item) {
					return true
				}
			}
		case map[string]any:
			if value, ok := typed["type"].(string); ok {
				switch value {
				case "input_text", "message":
				case "input_image", "input_file", "input_audio", "computer_screenshot", "item_reference", "computer_call", "computer_call_output":
					return true
				default:
					return true
				}
			} else {
				return true
			}
			for _, key := range []string{"image_url", "file_url", "image", "file", "audio"} {
				if value, ok := typed[key]; ok && value != nil && value != "" {
					return true
				}
			}
			for _, child := range typed {
				if visit(child) {
					return true
				}
			}
		}
		return false
	}
	return visit(value)
}

func buildResponsesFallbackUsage(c *gin.Context, info *relaycommon.RelayInfo, observer *responsesUsageObserver) *dto.Usage {
	completionTokens := observer.outputTokenCount(info.UpstreamModelName)
	usage := &dto.Usage{PromptTokens: info.GetEstimatePromptTokens(), CompletionTokens: completionTokens}
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	common.SetContextKey(c, constant.ContextKeyLocalCountTokens, true)
	return usage
}

func recordResponsesBillingEvidence(info *relaycommon.RelayInfo, observer *responsesUsageObserver, source string, eligible bool, ineligibleReason string, observedOutputTokens int) {
	if info == nil || observer == nil {
		return
	}
	_, output, reasoning, arguments, uncertain, normalTerminal, terminalFailure, authoritativeSeen, authoritativeValid, rejected := observer.finalize()
	cutoff, frozen := uint64(0), false
	if info.StreamStatus != nil {
		cutoff, frozen = info.StreamStatus.CutoffSequence()
	}
	if ineligibleReason == "" {
		switch {
		case uncertain:
			ineligibleReason = "usage_inspection_uncertain"
		case terminalFailure:
			ineligibleReason = "provider_terminal_failure"
		case normalTerminal:
			ineligibleReason = "normal_terminal"
		case !frozen:
			ineligibleReason = "cutoff_not_frozen"
		case rejected != "":
			ineligibleReason = "authoritative_usage_rejected"
		default:
			ineligibleReason = "eligibility_gate_failed"
		}
	}
	categories := make([]string, 0, 3)
	if output != "" {
		categories = append(categories, "output_text")
	}
	if reasoning != "" {
		categories = append(categories, "reasoning")
	}
	if arguments != "" {
		categories = append(categories, "tool_arguments")
	}
	fallbackCachePolicy := ""
	if source == "local" && eligible {
		fallbackCachePolicy = "ordinary_input"
	}
	info.ClientGoneBillingEvidence = &relaycommon.ClientGoneBillingEvidence{
		UsageSource:                      source,
		TerminalEventSeen:                normalTerminal,
		ReceivedDataEvents:               info.ReceivedResponseCount,
		EstimatedPromptTokens:            info.GetEstimatePromptTokens(),
		ObservedOutputTokens:             observedOutputTokens,
		ObservedOutputCategories:         categories,
		InputTokenizerModel:              info.OriginModelName,
		OutputTokenizerModel:             info.UpstreamModelName,
		AuthoritativeUsageSeen:           authoritativeSeen,
		AuthoritativeUsageValid:          authoritativeValid,
		AuthoritativeUsageRejectReason:   rejected,
		UsageInspectionComplete:          !uncertain && info.StreamStatus != nil && info.StreamStatus.InspectionComplete(),
		UsageInspectionUncertaintyReason: observer.uncertaintyReason,
		FallbackEligibilityResult:        eligible,
		FallbackIneligibleReason:         ineligibleReason,
		FallbackCachePolicy:              fallbackCachePolicy,
		CutoffSequence:                   cutoff,
	}
}

func OaiResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	// read response body
	var responsesResponse dto.OpenAIResponsesResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	err = common.Unmarshal(responseBody, &responsesResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	if responsesResponse.HasImageGenerationCall() {
		c.Set("image_generation_call", true)
		c.Set("image_generation_call_quality", responsesResponse.GetQuality())
		c.Set("image_generation_call_size", responsesResponse.GetSize())
	}

	// 写入新的 response body
	service.IOCopyBytesGracefully(c, resp, responseBody)

	// compute usage
	usage := dto.Usage{}
	if responsesResponse.Usage != nil {
		usage.PromptTokens = responsesResponse.Usage.InputTokens
		usage.CompletionTokens = responsesResponse.Usage.OutputTokens
		usage.TotalTokens = responsesResponse.Usage.TotalTokens
		if responsesResponse.Usage.InputTokensDetails != nil {
			usage.PromptTokensDetails.CachedTokens = responsesResponse.Usage.InputTokensDetails.CachedTokens
			usage.PromptTokensDetails.CacheWriteTokens = responsesResponse.Usage.InputTokensDetails.CacheWriteTokens
			usage.PromptTokensDetails.CacheCreationInputTokens = responsesResponse.Usage.InputTokensDetails.CacheCreationInputTokens
		}
	}
	if info == nil || info.ResponsesUsageInfo == nil || info.ResponsesUsageInfo.BuiltInTools == nil {
		return &usage, nil
	}
	// 解析 Tools 用量
	for _, tool := range responsesResponse.Tools {
		buildToolinfo, ok := info.ResponsesUsageInfo.BuiltInTools[common.Interface2String(tool["type"])]
		if !ok || buildToolinfo == nil {
			logger.LogError(c, fmt.Sprintf("BuiltInTools not found for tool type: %v", tool["type"]))
			continue
		}
		buildToolinfo.CallCount++
	}
	return &usage, nil
}

func OaiResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		logger.LogError(c, "invalid response or response body")
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	defer service.CloseResponseBodyGracefully(resp)

	var usage = &dto.Usage{}
	observer := newResponsesUsageObserver()
	if info != nil {
		info.StreamObserver = observer
	}

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {

		// 检查当前数据是否包含 completed 状态和 usage 信息
		var streamResponse dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResponse); err != nil {
			logger.LogError(c, "failed to unmarshal stream response: "+err.Error())
			sr.Error(err)
			return
		}
		sendResponsesStreamData(c, streamResponse, data)
		switch streamResponse.Type {
		case "response.completed":
			if streamResponse.Response != nil {
				if streamResponse.Response.HasImageGenerationCall() {
					c.Set("image_generation_call", true)
					c.Set("image_generation_call_quality", streamResponse.Response.GetQuality())
					c.Set("image_generation_call_size", streamResponse.Response.GetSize())
				}
			}
		case dto.ResponsesOutputTypeItemDone:
			// 函数调用处理
			if streamResponse.Item != nil {
				switch streamResponse.Item.Type {
				case dto.BuildInCallWebSearchCall:
					if info != nil && info.ResponsesUsageInfo != nil && info.ResponsesUsageInfo.BuiltInTools != nil {
						if webSearchTool, exists := info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool != nil {
							webSearchTool.CallCount++
						}
					}
				}
			}
		}
	})

	authoritativeUsage, _, _, _, _, normalTerminal, terminalFailure, _, authoritativeValid, _ := observer.finalize()
	if info != nil && info.StreamStatus != nil && normalTerminal && info.StreamStatus.GetEndReason() == relaycommon.StreamEndReasonClientGone {
		info.StreamStatus.CorrectClientGoneToDone(normalTerminal)
	}
	cutoffFrozen := false
	if info != nil && info.StreamStatus != nil {
		_, cutoffFrozen = info.StreamStatus.CutoffSequence()
	}
	if !constant.ClientGoneFallbackEnabled || !cutoffFrozen {
		return observer.legacyFallbackUsage(info), nil
	}
	if authoritativeValid && authoritativeUsage != nil {
		usage = authoritativeUsage
		recordResponsesBillingEvidence(info, observer, "upstream", false, "", 0)
	} else if responsesFallbackEligible(info, observer) {
		usage = buildResponsesFallbackUsage(c, info, observer)
		recordResponsesBillingEvidence(info, observer, "local", true, "", usage.CompletionTokens)
	} else if terminalFailure {
		// An explicit provider failure must never use the legacy local output
		// estimate, even when the scanner later reports EOF.
		usage = &dto.Usage{}
		recordResponsesBillingEvidence(info, observer, "", false, "", 0)
	} else {
		// Preserve the existing local output fallback for requests outside the
		// guarded client-gone branch. This keeps compatibility channels unchanged.
		usage = observer.legacyFallbackUsage(info)
		if info != nil && info.StreamStatus != nil && info.StreamStatus.GetEndReason() == relaycommon.StreamEndReasonClientGone {
			source := ""
			if usage.TotalTokens > 0 {
				source = "local"
			}
			recordResponsesBillingEvidence(info, observer, source, false, "", usage.CompletionTokens)
		}
	}

	return usage, nil
}
