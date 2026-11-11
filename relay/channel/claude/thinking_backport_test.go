package claude

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeForwardsEveryBetaHeaderLine(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	c.Request.Header.Add("anthropic-beta", "first-beta")
	c.Request.Header.Add("anthropic-beta", "thinking-binding-controls-2026-08-01")
	headers := http.Header{}
	CommonClaudeHeadersOperation(c, &headers, &relaycommon.RelayInfo{})
	assert.Equal(t, "first-beta,thinking-binding-controls-2026-08-01", headers.Get("anthropic-beta"))
}
func TestClaudeCacheOnlyStreamPreservesZeroOrdinaryUsage(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatClaude, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "claude-opus-4-8"}}
	info.SetEstimatePromptTokens(1200)
	state := &ClaudeResponseInfo{Usage: &dto.Usage{}}
	for _, event := range []string{
		`{"type":"message_start","message":{"id":"msg_test","usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":100}}}`,
		`{"type":"message_delta","delta":{"stop_reason":"refusal"},"usage":{"output_tokens":0}}`,
		`{"type":"message_stop"}`,
	} {
		require.Nil(t, HandleStreamResponseData(c, info, state, event))
	}
	HandleStreamFinalResponse(c, info, state)
	assert.Zero(t, state.Usage.PromptTokens)
	assert.Zero(t, state.Usage.CompletionTokens)
	assert.Equal(t, 100, state.Usage.PromptTokensDetails.CachedCreationTokens)
	encoded, err := common.Marshal(state.Usage.BillingUsage)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"input_tokens":0`)
}

func TestClaudeRenderedThinkingRestoresOnlyAdaptiveBinding(t *testing.T) {
	var original dto.ClaudeRequest
	require.NoError(t, common.UnmarshalJsonStr(`{"thinking":{"type":"adaptive","block_binding":{"prefix_mismatch_behavior":"drop_block"}}}`, &original))
	for _, mode := range []string{"adaptive", "enabled", "between_tools"} {
		t.Run(mode, func(t *testing.T) {
			request := &dto.ClaudeRequest{Thinking: &dto.Thinking{Type: mode}}
			result, err := (&Adaptor{}).ConvertClaudeRequest(nil, &relaycommon.RelayInfo{Request: &original}, request)
			require.NoError(t, err)
			encoded, err := common.Marshal(result)
			require.NoError(t, err)
			if mode == "adaptive" {
				assert.Contains(t, string(encoded), `"block_binding":{"prefix_mismatch_behavior":"drop_block"}`)
			} else {
				assert.NotContains(t, string(encoded), "block_binding")
			}
		})
	}
	// A client that did not send a binding must not acquire one.
	request := &dto.ClaudeRequest{Thinking: &dto.Thinking{Type: "adaptive"}}
	_, err := (&Adaptor{}).ConvertClaudeRequest(nil, &relaycommon.RelayInfo{Request: &dto.ClaudeRequest{}}, request)
	require.NoError(t, err)
	assert.Empty(t, request.Thinking.BlockBinding)
}
