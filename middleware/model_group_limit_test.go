package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfiguredGroupRateLimitsAreIndependent(t *testing.T) {
	oldEnabled, oldDuration, oldTotal, oldSuccess, oldGroups := setting.ModelRequestRateLimitEnabled, setting.ModelRequestRateLimitDurationMinutes, setting.ModelRequestRateLimitCount, setting.ModelRequestRateLimitSuccessCount, setting.ModelRequestRateLimitGroup
	oldRedis := common.RedisEnabled
	t.Cleanup(func() {
		setting.ModelRequestRateLimitEnabled, setting.ModelRequestRateLimitDurationMinutes, setting.ModelRequestRateLimitCount, setting.ModelRequestRateLimitSuccessCount, setting.ModelRequestRateLimitGroup = oldEnabled, oldDuration, oldTotal, oldSuccess, oldGroups
		common.RedisEnabled = oldRedis
	})
	setting.ModelRequestRateLimitEnabled = true
	setting.ModelRequestRateLimitDurationMinutes = 60
	setting.ModelRequestRateLimitCount = 0
	setting.ModelRequestRateLimitSuccessCount = 1
	_, rdb := useRateLimitMiniRedis(t)
	for _, useRedis := range []bool{false, true} {
		t.Run(map[bool]string{false: "memory", true: "redis"}[useRedis], func(t *testing.T) {
			common.RedisEnabled = useRedis
			setting.ModelRequestRateLimitGroup = map[string][2]int{"claude": {0, 1}, "gpt": {0, 1}}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("id", 903721)
				common.SetContextKey(c, constant.ContextKeyTokenGroup, "claude,gpt")
				common.SetContextKey(c, constant.ContextKeyUsingGroup, c.Param("group"))
			}, ModelRequestRateLimit())
			router.GET("/:group", func(c *gin.Context) { c.Status(http.StatusOK) })
			send := func(group string) int {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/"+group, nil))
				return w.Code
			}
			assert.Equal(t, 200, send("claude"))
			assert.Equal(t, 429, send("claude"))
			assert.Equal(t, 200, send("gpt"))
			assert.Equal(t, 429, send("gpt"))
			// Groups without their own limit still share the user's global bucket.
			assert.Equal(t, 200, send("default"))
			assert.Equal(t, 429, send("other"))
			// Switching to total-request caps exercises the other counter family.
			setting.ModelRequestRateLimitGroup = map[string][2]int{"claude": {1, 100}, "gpt": {1, 100}}
			assert.Equal(t, 200, send("claude"))
			assert.Equal(t, 429, send("claude"))
			assert.Equal(t, 200, send("gpt"))
			assert.Equal(t, 429, send("gpt"))
			if useRedis {
				require.NoError(t, rdb.FlushDB(context.Background()).Err())
			}
		})
	}
}
