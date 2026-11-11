package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTieredClaudeCacheTotalWithoutCompleteSplit(t *testing.T) {
	for _, tc := range []struct {
		name                                  string
		total, five, hour, wantFive, wantHour int
	}{
		{"only total", 10, 0, 0, 10, 0}, {"partial split", 10, 3, 2, 8, 2}, {"split only", 0, 3, 2, 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := &dto.Usage{UsageSemantic: "anthropic", PromptTokens: 2, PromptTokensDetails: dto.InputTokenDetails{CachedCreationTokens: tc.total}, ClaudeCacheCreation5mTokens: tc.five, ClaudeCacheCreation1hTokens: tc.hour}
			params := BuildTieredTokenParams(usage, true, map[string]bool{"cc": true, "cc1h": true})
			assert.Equal(t, float64(tc.wantFive), params.CC)
			assert.Equal(t, float64(tc.wantHour), params.CC1h)
			assert.Equal(t, float64(2+tc.wantFive+tc.wantHour), params.Len)
		})
	}
}
func TestClaudeCacheOnlyUsageStillChargesQuota(t *testing.T) {
	for _, tc := range []struct {
		name        string
		read, write int
		want        int
	}{
		{"write", 0, 100, 250000}, {"read", 100, 0, 20000}, {"empty", 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatClaude, OriginModelName: "claude-opus-4-8", PriceData: types.PriceData{ModelRatio: 1, CacheRatio: 0.1, CacheCreationRatio: 1.25, CacheCreation5mRatio: 1.25, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}}}
			usage := &dto.Usage{UsageSemantic: "anthropic", PromptTokensDetails: dto.InputTokenDetails{CachedTokens: tc.read, CachedCreationTokens: tc.write}}
			summary := calculateTextQuotaSummary(ctx, info, usage)
			assert.Equal(t, tc.want, summary.Quota)
		})
	}
}
