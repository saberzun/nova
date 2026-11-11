package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAccessLoggerRedactsCredentialsWithoutChangingHandlerQuery(t *testing.T) {
	old := gin.DefaultWriter
	var out bytes.Buffer
	gin.DefaultWriter = &out
	t.Cleanup(func() { gin.DefaultWriter = old })
	router := gin.New()
	SetUpLogger(router)
	router.GET("/v1/models", func(c *gin.Context) { assert.Equal(t, "fixture-secret", c.Query("api_key")); c.Status(200) })
	router.GET("/api/oauth/callback", func(c *gin.Context) { assert.Equal(t, "fixture-code", c.Query("code")); c.Status(200) })
	for _, path := range []string{"/v1/models?api_key=fixture-secret&beta=true", "/api/oauth/callback?code=fixture-code&state=fixture-state"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	assert.NotContains(t, out.String(), "fixture-secret")
	assert.NotContains(t, out.String(), "fixture-code")
	assert.NotContains(t, out.String(), "fixture-state")
	assert.Contains(t, out.String(), "beta=true")
}
