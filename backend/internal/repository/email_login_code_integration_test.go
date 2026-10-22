//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// Fork-local D85: exercise authentication Lua against the harness's real Redis.
func TestEmailLoginCodeRealRedis(t *testing.T) {
	cache := NewEmailCache(integrationRedis).(service.EmailLoginCodeCache)
	ctx := context.Background()
	email := fmt.Sprintf("login-%d@example.com", time.Now().UnixNano())
	ok, err := cache.ReserveEmailLoginCode(ctx, email, "digest")
	require.NoError(t, err)
	require.True(t, ok)
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := cache.ConsumeEmailLoginCode(ctx, email, "digest")
			if err == nil && ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, wins.Load())
	ok, err = cache.ReserveEmailLoginCode(ctx, email, "replacement")
	require.NoError(t, err)
	require.False(t, ok)
	email = "attempts-" + email
	ok, err = cache.ReserveEmailLoginCode(ctx, email, "digest")
	require.NoError(t, err)
	require.True(t, ok)
	for i := 0; i < 5; i++ {
		ok, err = cache.ConsumeEmailLoginCode(ctx, email, "wrong")
		require.NoError(t, err)
		require.False(t, ok)
	}
	ok, err = cache.ConsumeEmailLoginCode(ctx, email, "digest")
	require.NoError(t, err)
	require.False(t, ok)
	email = "expiry-" + email
	ok, err = cache.ReserveEmailLoginCode(ctx, email, "digest")
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, integrationRedis.PExpireAt(ctx, emailLoginCodeKey(email), time.Now().Add(-time.Second)).Err())
	ok, err = cache.ConsumeEmailLoginCode(ctx, email, "digest")
	require.NoError(t, err)
	require.False(t, ok)
}
