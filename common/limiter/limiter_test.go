package limiter

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiterRecoversScriptFlushWithoutResettingBucket(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	sha, err := client.ScriptLoad(ctx, rateLimitScript).Result()
	require.NoError(t, err)
	rl := &RedisLimiter{client: client, limitScriptSHA: sha}
	allowed, err := rl.Allow(ctx, "flush-bucket", WithCapacity(2), WithRate(0))
	require.NoError(t, err)
	assert.True(t, allowed)
	require.NoError(t, client.ScriptFlush(ctx).Err())
	allowed, err = rl.Allow(ctx, "flush-bucket", WithCapacity(2), WithRate(0))
	require.NoError(t, err)
	assert.True(t, allowed)
	allowed, err = rl.Allow(ctx, "flush-bucket", WithCapacity(2), WithRate(0))
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestLimiterReportsNonScriptRedisErrors(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { client.Close() })
	ctx := context.Background()
	sha, err := client.ScriptLoad(ctx, rateLimitScript).Result()
	require.NoError(t, err)
	require.NoError(t, client.Set(ctx, "wrong-type-bucket", "not-a-hash", 0).Err())
	allowed, err := (&RedisLimiter{client: client, limitScriptSHA: sha}).Allow(ctx, "wrong-type-bucket")
	require.Error(t, err)
	assert.False(t, allowed)
}
