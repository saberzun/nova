package aws

import (
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBedrockRetainsThinkingBindingAndNormalizesAllBetaLines(t *testing.T) {
	header := http.Header{}
	header.Add("anthropic-beta", "first, second, ")
	header.Add("anthropic-beta", " third")
	request, err := formatRequest(strings.NewReader(`{"thinking":{"type":"adaptive","block_binding":{"prefix_mismatch_behavior":"drop_block"},"future_field":false}}`), header)
	require.NoError(t, err)
	assert.JSONEq(t, `["first","second","third"]`, string(request.AnthropicBeta))
	body, err := common.Marshal(request.Thinking)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"adaptive","block_binding":{"prefix_mismatch_behavior":"drop_block"},"future_field":false}`, string(body))
}
