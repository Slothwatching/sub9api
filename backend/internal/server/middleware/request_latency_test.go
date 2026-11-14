package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/requestlatency"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Local mock smoke: real HTTP transport, ingress middleware, 429 outer retry,
// lifecycle frames, visible content and actual downstream Flush.
func TestRequestLatencyMockRetrySmoke(t *testing.T) {
	for _, protocol := range []string{"responses", "chat"} {
		t.Run(protocol, func(t *testing.T) {
			var attempts atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attempts.Add(1) == 1 {
					w.WriteHeader(429)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				preamble := `{"type":"response.created"}`
				content := `{"type":"response.output_text.delta","delta":"hello"}`
				if protocol == "chat" {
					preamble = `{"choices":[{"delta":{"role":"assistant"}}]}`
					content = `{"choices":[{"delta":{"content":"hello"}}]}`
				}
				fmt.Fprintf(w, "data: %s\n\n", preamble)
				w.(http.Flusher).Flush()
				time.Sleep(5 * time.Millisecond)
				fmt.Fprintf(w, "data: %s\n\n", content)
				w.(http.Flusher).Flush()
			}))
			defer upstream.Close()
			var got requestlatency.Snapshot
			router := gin.New()
			router.Use(WithRequestLatency(func(c *gin.Context) {
				require.NotNil(t, requestlatency.From(c.Request.Context()), "clock precedes auth and body read")
				time.Sleep(5 * time.Millisecond) // ingress/auth/body overhead
			}))
			router.POST("/test", func(c *gin.Context) {
				timing := requestlatency.From(c.Request.Context())
				stop := timing.Segment(requestlatency.UserQueue)
				time.Sleep(5 * time.Millisecond)
				stop()
				for i := 0; i < 2; i++ {
					end := timing.BeginAttempt()
					req, err := http.NewRequestWithContext(timing.Trace(c.Request.Context()), http.MethodGet, upstream.URL, nil)
					require.NoError(t, err)
					resp, err := upstream.Client().Do(req)
					require.NoError(t, err)
					if resp.StatusCode == 429 {
						resp.Body.Close()
						end(true)
						continue
					}
					c.Header("Content-Type", "text/event-stream")
					body := requestlatency.ObserveBody(resp.Body, timing, true)
					// Copy in small fragments to exercise parsing across downstream writes.
					_, err = io.CopyBuffer(c.Writer, struct{ io.Reader }{body}, make([]byte, 7))
					require.NoError(t, err)
					require.Nil(t, timing.Snapshot().ClientFirstContentMs, "content written but not flushed yet")
					c.Writer.Flush()
					body.Close()
					end(false)
				}
				timing.Mark(requestlatency.UserDuration)
				got = timing.Snapshot()
			})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/test", nil))
			require.Equal(t, 200, rec.Code)
			require.Equal(t, int32(2), attempts.Load())
			require.Equal(t, 2, *got.AttemptCount)
			require.NotNil(t, got.LastFailedAttemptMs)
			require.NotNil(t, got.UpstreamConnectMs)
			require.NotNil(t, got.UpstreamFirstByteMs)
			require.NotNil(t, got.FirstDataMs)
			require.NotNil(t, got.FirstSemanticMs)
			require.NotNil(t, got.FirstVisibleContentMs)
			require.GreaterOrEqual(t, *got.ClientFirstContentMs, 10)
			require.GreaterOrEqual(t, *got.ClientFirstContentMs, *got.FirstVisibleContentMs)
			require.GreaterOrEqual(t, *got.UserDurationMs, *got.ClientFirstContentMs)
			t.Logf("local %s mock: attempts=%d ingress_to_flush_ms=%d total_ms=%d", protocol, *got.AttemptCount, *got.ClientFirstContentMs, *got.UserDurationMs)
		})
	}
}
func TestRequestLatencySkipsLifecycleFlushAndBufferedJSON(t *testing.T) {
	for _, visible := range []bool{false, true} {
		t.Run(fmt.Sprint(visible), func(t *testing.T) {
			var got requestlatency.Snapshot
			r := gin.New()
			r.Use(WithRequestLatency(func(c *gin.Context) {}))
			r.GET("/", func(c *gin.Context) {
				if visible {
					c.JSON(200, gin.H{"choices": []any{gin.H{"message": gin.H{"content": "ok"}}}})
				} else {
					c.Header("Content-Type", "text/event-stream")
					c.Writer.WriteString("data: {\"type\":\"response.created\"}\n\n")
					c.Writer.Flush()
				}
				got = requestlatency.From(c.Request.Context()).Snapshot()
			})
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			if visible {
				require.NotNil(t, got.ClientFirstContentMs)
			} else {
				require.Nil(t, got.ClientFirstContentMs)
			}
		})
	}
	ctx, timing := requestlatency.New(context.Background())
	require.Same(t, timing, requestlatency.From(ctx))
}
