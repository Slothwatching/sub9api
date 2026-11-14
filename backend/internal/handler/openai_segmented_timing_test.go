//go:build unit

package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/requestlatency"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// Runs ingress middleware, the real Responses handler/scheduler/Forward,
// doOpenAIUpstream body observation, submit snapshot and RecordUsage. Only auth,
// account storage, usage storage and the upstream HTTP endpoint are fixtures;
// no real provider credentials, billing database or scheduler cache is needed.
func TestOpenAISegmentedTimingHandlerRetryUsage(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"type":"rate_limit_error","message":"fixture retry"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"resp_fixture","object":"response","model":"gpt-5.1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}`)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	transport := &nativeImageRouteUpstream{target: target, client: upstream.Client()}
	h := newOpenAIResponsesFailoverTestHandler(t, transport)
	cfg := h.cfg
	cfg.Default.RateMultiplier = 1
	cfg.Security.URLAllowlist.Enabled = false
	accounts := []service.Account{}
	for i := 1; i <= 2; i++ {
		accounts = append(accounts, service.Account{ID: int64(i), Name: "fixture", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Priority: i, Credentials: map[string]any{"api_key": "fixture", "base_url": "https://fixture.example"}, Extra: map[string]any{"openai_passthrough": true}})
	}
	logs := &openAIWSUsageHandlerUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	repo := openAIImagesFailoverAccountRepo{accounts: accounts}
	h.gatewayService = service.NewOpenAIGatewayService(repo, logs, nil, nil, nil, nil, nil, cfg, nil, nil, service.NewBillingService(cfg, nil), nil, nil, transport, &service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil)
	c, rec := newOpenAIResponsesFailoverTestContext(t, nil)
	middleware.WithRequestLatency(h.Responses)(c)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, int32(2), calls.Load())
	select {
	case log := <-logs.created:
		got := log.Snapshot
		require.Equal(t, 2, *got.AttemptCount)
		require.NotNil(t, got.LastFailedAttemptMs)
		require.NotNil(t, got.LastAttemptMs)
		require.NotNil(t, got.UpstreamConnectMs)
		require.NotNil(t, got.UpstreamFirstByteMs)
		require.NotNil(t, got.FirstDataMs)
		require.NotNil(t, got.FirstSemanticMs)
		require.NotNil(t, got.FirstVisibleContentMs)
		require.NotNil(t, got.ClientFirstContentMs)
		require.NotNil(t, got.UserDurationMs)
		require.GreaterOrEqual(t, *got.UserDurationMs, *got.ClientFirstContentMs)
		require.Equal(t, requestlatency.From(c.Request.Context()).Snapshot(), got)
		require.False(t, rec.Flushed)
		require.NotNil(t, log.DurationMs, "legacy attempt duration remains independently recorded")
	case <-time.After(2 * time.Second):
		t.Fatal("usage snapshot not persisted")
	}
}

func TestOpenAISegmentedTimingWebSocketSelectionUsage(t *testing.T) {
	// Exercise the real handler context replacements, scheduler, AfterTurn and
	// usage persistence. Discarding ForkAt's context loses this selection mark.
	got := runOpenAIResponsesWebSocketUsageLogCase(t, openAIResponsesWSUsageLogCase{
		firstPayload: `{"type":"response.create","model":"gpt-5.4","stream":false}`,
	})
	require.NotNil(t, got.log.Snapshot.AccountSelectionMs)
}

// Covers the same turn registry and submit path used by AfterTurn. Explicit
// starts avoid timing thresholds or sleeps while detecting handshake reuse and
// first-wins marks shared between turns. Native WS content remains unmeasured.
func TestOpenAISegmentedTimingWebSocketTurnsSubmit(t *testing.T) {
	handshakeCtx, handshake := requestlatency.ForkAt(context.Background(), time.Now().Add(-time.Hour))
	handshake.Mark(requestlatency.UserDuration)
	handshake.Mark(requestlatency.FirstVisibleContent)
	original := handshake.Snapshot()
	var turns openAIWSTurnTimings
	h := &OpenAIGatewayHandler{}
	var snapshots []requestlatency.Snapshot
	for turn, age := range []time.Duration{10 * time.Second, time.Second} {
		turn++
		turns.begin(turn, time.Now().Add(-age))
		parent := turns.context(handshakeCtx, turn)
		clock := requestlatency.From(parent)
		require.NotSame(t, handshake, clock)
		// Same context lookup and Segment as SelectAccountWithSchedulerForCapability.
		requestlatency.From(parent).Segment(requestlatency.AccountSelection)()
		clock.Mark(requestlatency.UpstreamFirstByte)
		result := &service.OpenAIForwardResult{}
		h.submitOpenAIUsageRecordTask(parent, result, func(context.Context) {
			require.NotNil(t, result.Latency.UserDurationMs)
			snapshots = append(snapshots, result.Latency)
		})
		turns.finish(turn)
	}
	require.Len(t, snapshots, 2)
	require.Greater(t, *snapshots[0].UserDurationMs, *snapshots[1].UserDurationMs+5000)
	require.Greater(t, *snapshots[0].UpstreamFirstByteMs, *snapshots[1].UpstreamFirstByteMs+5000)
	for _, got := range snapshots {
		require.NotNil(t, got.AccountSelectionMs)
		require.Less(t, *got.UserDurationMs, *original.UserDurationMs)
		require.Nil(t, got.FirstVisibleContentMs)
		require.Nil(t, got.ClientFirstContentMs)
	}
	require.Equal(t, original, handshake.Snapshot())
	require.Empty(t, turns.clocks)
}

func TestOpenAISegmentedTimingWebSocketLaterTurnRetry(t *testing.T) {
	parent, handshake := requestlatency.ForkAt(context.Background(), time.Now().Add(-time.Hour))
	var turns openAIWSTurnTimings
	turns.begin(1, time.Now().Add(-10*time.Second))
	first := requestlatency.From(turns.context(parent, 1))
	first.Mark(requestlatency.UserDuration)
	turns.finish(1)
	turns.begin(2, time.Now().Add(-time.Second))
	second := requestlatency.From(turns.context(parent, 2))
	second.Mark(requestlatency.UpstreamFirstByte)
	turns.finish(2) // failed attempt; service will number the replay as turn 1
	retryClock := turns.current()
	// The outer failover loop must reinstall the latest logical-turn clock
	// before selecting the replacement account, which seeds proxy turn 1.
	selectionCtx := retryClock.Context(parent)
	requestlatency.From(selectionCtx).Segment(requestlatency.AccountSelection)()
	var replacementAccountTurns openAIWSTurnTimings
	replacementAccountTurns.seed(retryClock)
	replacementAccountTurns.begin(1, time.Now()) // admission must retain replay clock
	retryCtx := replacementAccountTurns.context(parent, 1)
	require.Same(t, second, requestlatency.From(retryCtx))
	require.NotSame(t, first, requestlatency.From(retryCtx))
	result := &service.OpenAIForwardResult{}
	(&OpenAIGatewayHandler{}).submitOpenAIUsageRecordTask(retryCtx, result, nil)
	require.NotNil(t, result.Latency.UpstreamFirstByteMs)
	require.NotNil(t, result.Latency.AccountSelectionMs)
	require.Nil(t, first.Snapshot().AccountSelectionMs)
	require.Less(t, *result.Latency.UserDurationMs, *first.Snapshot().UserDurationMs)
	replacementAccountTurns.finish(1)
	replacementAccountTurns.begin(2, time.Now()) // a subsequent distinct turn
	require.NotSame(t, second, requestlatency.From(replacementAccountTurns.context(parent, 2)))
	require.Nil(t, requestlatency.From(replacementAccountTurns.context(parent, 2)).Snapshot().UserDurationMs)
	require.Equal(t, requestlatency.Snapshot{}, handshake.Snapshot())
}
