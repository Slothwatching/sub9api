package requestlatency

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimingIngressSurvivesRetryAndMissingEvents(t *testing.T) {
	ctx, timing := New(context.Background())
	timing.start = time.Now().Add(-time.Second)
	retryCtx, same := New(ctx)
	require.Same(t, timing, same)
	require.Same(t, timing, From(context.WithoutCancel(retryCtx)))
	require.Equal(t, Snapshot{}, timing.Snapshot())
	stop := timing.Segment(UserQueue)
	stop()
	timing.Add(AccountQueue, 3*time.Millisecond)
	timing.Add(AccountQueue, 4*time.Millisecond)
	first := timing.BeginAttempt()
	first(true)
	second := same.BeginAttempt()
	second(false)
	timing.Mark(FirstSemantic)
	timing.Mark(UserDuration)
	got := timing.Snapshot()
	require.Equal(t, 2, *got.AttemptCount)
	require.NotNil(t, got.LastFailedAttemptMs)
	require.NotNil(t, got.LastAttemptMs)
	require.NotNil(t, got.UserQueueMs)
	require.Equal(t, 7, *got.AccountQueueMs)
	require.GreaterOrEqual(t, *got.FirstSemanticMs, 1000)
	require.GreaterOrEqual(t, *got.UserDurationMs, 1000)
	require.Nil(t, got.FirstVisibleContentMs)
	require.Nil(t, got.ClientFirstContentMs)
	require.Nil(t, got.UpstreamTLSMs)
	// First observations and detached snapshots do not drift after retries.
	old := *got.FirstSemanticMs
	timing.start = timing.start.Add(-time.Second)
	timing.Mark(FirstSemantic)
	require.Equal(t, old, *timing.Snapshot().FirstSemanticMs)
	third := timing.BeginAttempt()
	third(false)
	require.Equal(t, 2, *got.AttemptCount)
}
func TestTimingNilAndTrace(t *testing.T) {
	var missing *Timing
	missing.Mark(FirstData)
	missing.Add(UserQueue, time.Second)
	missing.Segment(UserQueue)()
	missing.BeginAttempt()(false)
	require.Equal(t, Snapshot{}, missing.Snapshot())
	ctx, timing := New(context.Background())
	called := false
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{GotFirstResponseByte: func() { called = true }})
	trace := httptrace.ContextClientTrace(timing.Trace(ctx))
	trace.GetConn("mock")
	trace.GotConn(httptrace.GotConnInfo{Reused: true})
	trace.GotFirstResponseByte()
	require.True(t, called)
	require.NotNil(t, timing.Snapshot().UpstreamConnectMs)
	require.NotNil(t, timing.Snapshot().UpstreamFirstByteMs)
	require.Nil(t, timing.Snapshot().UpstreamTLSMs)
}
func TestTimingProtocolEvents(t *testing.T) {
	cases := []struct {
		name, payload     string
		semantic, visible bool
	}{
		{"chat role", `{"choices":[{"delta":{"role":"assistant","content":""}}]}`, false, false},
		{"chat tool name", `{"choices":[{"delta":{"tool_calls":[{"function":{"name":"test","arguments":""}}]}}]}`, true, false},
		{"responses tool name", `{"type":"response.output_item.added","item":{"type":"function_call","name":"test","arguments":""}}`, true, false},
		{"chat tool structure", `{"choices":[{"delta":{"tool_calls":[{"id":"call_test","function":{"name":"","arguments":""}}]}}]}`, true, false},
		{"response typed usage", `{"type":"response.usage","usage":{}}`, false, false},
		{"chat usage", `{"choices":[],"usage":{"total_tokens":3}}`, false, false},
		{"chat finish", `{"choices":[{"delta":{},"finish_reason":"stop"}]}`, false, false},
		{"chat empty parts", `{"choices":[{"delta":{"content":[{"type":"text","text":""}]}}]}`, false, false},
		{"chat text parts", `{"choices":[{"delta":{"content":[{"type":"text","text":"hi"}]}}]}`, true, true},
		{"chat content", `{"choices":[{"delta":{"content":"hi"}}]}`, true, true},
		{"chat tool", `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{"}}]}}]}`, true, true},
		{"created", `{"type":"response.created"}`, false, false},
		{"queue", `{"type":"response.queued"}`, false, false},
		{"progress", `{"type":"response.in_progress"}`, false, false},
		{"metadata", `{"type":"codex.response.metadata"}`, false, false},
		{"keepalive", `{"type":"keepalive"}`, false, false},
		{"usage", `{"usage":{"total_tokens":3}}`, false, false},
		{"lifecycle", `{"type":"response.completed","response":{"usage":{"total_tokens":3}}}`, false, false},
		{"failed", `{"type":"response.failed"}`, false, false},
		{"structural", `{"type":"response.output_item.added","item":{"type":"message","role":"assistant","content":[]}}`, true, false},
		{"response text", `{"type":"response.output_text.delta","delta":"hi"}`, true, true},
		{"response tool", `{"type":"response.function_call_arguments.delta","delta":"{"}`, true, true},
		{"empty delta", `{"type":"response.output_text.delta","delta":""}`, true, false},
		{"buffered", `{"output":[{"type":"message","content":[{"type":"output_text","text":"hi"}]}]}`, true, true},
		{"image", `{"data":[{"b64_json":"fake"}]}`, true, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, v := Classify([]byte(tt.payload))
			require.Equal(t, tt.semantic, s)
			require.Equal(t, tt.visible, v)
		})
	}
}
func TestTimingObserverFragmentsAndFlushSeparation(t *testing.T) {
	for _, protocol := range []string{"chat", "responses"} {
		t.Run(protocol, func(t *testing.T) {
			_, timing := New(context.Background())
			observer := Observer{Timing: timing, SSE: true}
			preamble := `{"type":"response.created"}`
			content := `{"type":"response.output_text.delta","delta":"hello"}`
			if protocol == "chat" {
				preamble = `{"choices":[{"delta":{"role":"assistant"}}]}`
				content = `{"choices":[{"delta":{"content":"hello"}}]}`
			}
			for _, b := range []byte("data: " + preamble + "\r\n\r\n") {
				observer.Feed([]byte{b})
			}
			require.NotNil(t, timing.Snapshot().FirstDataMs)
			require.Nil(t, timing.Snapshot().FirstSemanticMs)
			observer.Feed([]byte(": heartbeat\n\ndata: " + content + "\n\n"))
			got := timing.Snapshot()
			require.NotNil(t, got.FirstSemanticMs)
			require.NotNil(t, got.FirstVisibleContentMs)
			require.Nil(t, got.ClientFirstContentMs, "observing content is not proof of client flush")
		})
	}
	_, timing := New(context.Background())
	o := Observer{Timing: timing, SSE: true}
	o.Feed([]byte("event: response.output_text.delta\ndata: {\"delta\":\"hi\"}\n\n"))
	require.NotNil(t, timing.Snapshot().FirstVisibleContentMs)
}
func TestTimingObserverBoundedAndUsageOnly(t *testing.T) {
	_, timing := New(context.Background())
	o := Observer{Timing: timing, SSE: true}
	o.Feed([]byte("data: " + strings.Repeat("x", maxObservation+1) + "\n\n"))
	require.LessOrEqual(t, cap(o.buf), maxObservation)
	o.Feed([]byte("data: {\"choices\":[],\"usage\":{}}\n\n"))
	o.End()
	require.Nil(t, timing.Snapshot().FirstVisibleContentMs)
	require.Nil(t, timing.Snapshot().FirstSemanticMs)
	o.Feed([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	require.NotNil(t, timing.Snapshot().FirstVisibleContentMs)
}

func TestTimingObserverOversizedEventRecovery(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n"} {
		for _, split := range []bool{false, true} {
			for _, content := range []string{
				`event: image_generation.completed` + ending + `data: {"b64_json":"ok"}`,
				`event: response.output_text.delta` + ending + `data: {"delta":"ok"}`,
			} {
				t.Run(fmt.Sprintf("ending=%q/split=%t/%s", ending, split, content[:20]), func(t *testing.T) {
					_, clock := New(context.Background())
					o := Observer{Timing: clock, SSE: true}
					o.Feed([]byte("event: response.output_text.delta" + ending))
					o.Feed([]byte("data: " + strings.Repeat("x", maxObservation+1) + ending))
					require.True(t, o.dropping, "nonempty CRLF line must not end the dropped event")
					o.Feed([]byte(`data: {"delta":"still inside oversized event"}` + ending))
					require.True(t, o.dropping)
					require.Nil(t, clock.Snapshot().FirstVisibleContentMs)
					if split {
						// CR and LF of the blank delimiter arrive in separate calls.
						o.Feed([]byte(strings.TrimSuffix(ending, "\n")))
						require.True(t, o.dropping, "an incomplete line cannot recover")
						o.Feed([]byte("\n"))
					} else {
						o.Feed([]byte(ending))
					}
					require.False(t, o.dropping)
					require.Empty(t, o.event)
					// No event name or type: the old delta event must not leak.
					o.Feed([]byte(`data: {"delta":"untyped"}` + ending + ending))
					require.Nil(t, clock.Snapshot().FirstSemanticMs)
					require.Nil(t, clock.Snapshot().FirstVisibleContentMs)
					o.Feed([]byte(content + ending + ending))
					require.True(t, o.Visible, "a later qualifying event must still be observed")
					require.NotNil(t, clock.Snapshot().FirstVisibleContentMs)
					require.LessOrEqual(t, cap(o.buf), maxObservation)
				})
			}
		}
	}
}

func TestTimingActualTLSHooks(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) }))
	defer upstream.Close()
	ctx, clock := New(context.Background())
	req, err := http.NewRequestWithContext(clock.Trace(ctx), http.MethodGet, upstream.URL, nil)
	require.NoError(t, err)
	resp, err := upstream.Client().Do(req)
	require.NoError(t, err)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	got := clock.Snapshot()
	require.NotNil(t, got.UpstreamTLSMs)
	require.NotNil(t, got.UpstreamConnectMs)
	require.NotNil(t, got.UpstreamFirstByteMs)
	require.GreaterOrEqual(t, *got.UpstreamConnectMs, *got.UpstreamTLSMs)
}
