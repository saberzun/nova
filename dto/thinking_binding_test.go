package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThinkingPreservesBindingAndUnknownFields(t *testing.T) {
	for _, raw := range []string{
		`{"type":"adaptive","block_binding":{"prefix_mismatch_behavior":"drop_block"},"future_field":{"enabled":false},"budget_tokens":0}`,
		`{"type":"adaptive","display":"summarized"}`,
	} {
		var request ClaudeRequest
		require.NoError(t, common.UnmarshalJsonStr(`{"model":"claude-opus-5-5","thinking":`+raw+`}`, &request))
		copy, err := common.DeepCopy(&request)
		require.NoError(t, err)
		encoded, err := common.Marshal(copy.Thinking)
		require.NoError(t, err)
		assert.JSONEq(t, raw, string(encoded))
	}
}
func TestCacheCreationInputTokensAreCountedWithoutDuplication(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int
	}{
		{`{"cache_creation_input_tokens":9}`, 9},
		{`{"cache_creation_input_tokens":9,"cache_write_tokens":7,"cached_creation_tokens":8}`, 9},
		{`{"cache_creation_input_tokens":-9}`, 0},
	} {
		var d InputTokenDetails
		require.NoError(t, common.UnmarshalJsonStr(tc.raw, &d))
		assert.Equal(t, tc.want, d.CacheCreationTokensTotal())
	}
}
