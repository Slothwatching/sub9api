package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestEmailLoginCodeSingleUseConcurrent(t *testing.T) {
	base, _, _ := newMiniredisEmailCache(t)
	cache, implementsLoginCache := base.(service.EmailLoginCodeCache)
	require.True(t, implementsLoginCache)
	ctx := context.Background()
	ok, err := cache.ReserveEmailLoginCode(ctx, "User@example.com", "digest")
	require.NoError(t, err)
	require.True(t, ok)
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := cache.ConsumeEmailLoginCode(ctx, " user@example.com ", "digest")
			if err == nil && ok {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, successes.Load())
	ok, err = cache.ReserveEmailLoginCode(ctx, "user@example.com", "replacement")
	require.NoError(t, err)
	require.False(t, ok, "consumption must not reset cooldown")
}

func TestEmailLoginCodeAttemptExpiryAndIsolation(t *testing.T) {
	base, mr, _ := newMiniredisEmailCache(t)
	cache, implementsLoginCache := base.(service.EmailLoginCodeCache)
	require.True(t, implementsLoginCache)
	ctx := context.Background()
	require.NoError(t, base.SetVerificationCode(ctx, "user@example.com", &service.VerificationCodeData{Code: "123456"}, time.Hour))
	ok, err := cache.ConsumeEmailLoginCode(ctx, "user@example.com", "123456")
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = cache.ReserveEmailLoginCode(ctx, "user@example.com", "digest")
	require.NoError(t, err)
	require.True(t, ok)
	for i := 0; i < 5; i++ {
		ok, err = cache.ConsumeEmailLoginCode(ctx, "user@example.com", "wrong")
		require.NoError(t, err)
		require.False(t, ok)
	}
	ok, err = cache.ConsumeEmailLoginCode(ctx, "user@example.com", "digest")
	require.NoError(t, err)
	require.False(t, ok)
	mr.FastForward(time.Minute)
	ok, err = cache.ReserveEmailLoginCode(ctx, "user@example.com", "fresh")
	require.NoError(t, err)
	require.True(t, ok)
	mr.FastForward(5 * time.Minute)
	ok, err = cache.ConsumeEmailLoginCode(ctx, "user@example.com", "fresh")
	require.NoError(t, err)
	require.False(t, ok)
	data, err := base.GetVerificationCode(ctx, "user@example.com")
	require.NoError(t, err)
	require.Equal(t, "123456", data.Code)
}

func TestEmailLoginCodeHourlyLimitAndConcurrentSends(t *testing.T) {
	base, mr, _ := newMiniredisEmailCache(t)
	cache, implementsLoginCache := base.(service.EmailLoginCodeCache)
	require.True(t, implementsLoginCache)
	ctx := context.Background()
	for round := 0; round < 5; round++ {
		var wins atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ok, err := cache.ReserveEmailLoginCode(ctx, "u@example.com", "digest")
				if err == nil && ok {
					wins.Add(1)
				}
			}()
		}
		wg.Wait()
		require.EqualValues(t, 1, wins.Load())
		mr.FastForward(time.Minute)
	}
	ok, err := cache.ReserveEmailLoginCode(ctx, "u@example.com", "sixth")
	require.NoError(t, err)
	require.False(t, ok)
	mr.FastForward(time.Hour)
	ok, err = cache.ReserveEmailLoginCode(ctx, "u@example.com", "next-hour")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEmailLoginCodeCacheFailure(t *testing.T) {
	base, mr, _ := newMiniredisEmailCache(t)
	cache, implementsLoginCache := base.(service.EmailLoginCodeCache)
	require.True(t, implementsLoginCache)
	mr.Close()
	_, err := cache.ReserveEmailLoginCode(context.Background(), "u@example.com", "digest")
	require.Error(t, err)
	ok, err := cache.ConsumeEmailLoginCode(context.Background(), "u@example.com", "digest")
	require.Error(t, err)
	require.False(t, ok)
}
