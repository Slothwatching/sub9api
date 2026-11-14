package middleware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

// Models Ops capture/keepalive wrappers: update their own state under a lock,
// then unlock before calling the next writer. latencyWriter serializes the sink.
type unlockedCaptureWriter struct {
	gin.ResponseWriter
	mu     sync.Mutex
	writes int
}

func (w *unlockedCaptureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.writes++
	w.mu.Unlock()
	return w.ResponseWriter.Write(p)
}
func (w *unlockedCaptureWriter) Flush() {
	w.mu.Lock()
	w.writes++
	w.mu.Unlock()
	w.ResponseWriter.Flush()
}

func TestRequestLatencyConcurrentWrapperWriteFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	_, clock := requestlatency.New(context.Background())
	latency := &latencyWriter{ResponseWriter: c.Writer, observer: requestlatency.Observer{Timing: clock, Client: true}}
	outer := &unlockedCaptureWriter{ResponseWriter: latency}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = outer.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n"))
				_, _ = outer.WriteString(": keepalive\n\n")
				outer.Flush()
			}
		}()
	}
	wg.Wait()
	require.NotNil(t, clock.Snapshot().ClientFirstContentMs)
	require.Equal(t, 1600, outer.writes)
}

type stringSinkWriter struct{ gin.ResponseWriter }

func (w *stringSinkWriter) WriteString(s string) (int, error) { return len(s), nil }
func (w *stringSinkWriter) Write(p []byte) (int, error)       { return len(p), nil }

func TestRequestLatencyStoppedWriteStringDoesNotCopy(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	_, clock := requestlatency.New(context.Background())
	writer := &latencyWriter{ResponseWriter: &stringSinkWriter{c.Writer}, observer: requestlatency.Observer{Timing: clock, Client: true}}
	_, err := writer.WriteString("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
	require.NoError(t, err)
	payload := strings.Repeat("A", 2<<20)
	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = writer.WriteString(payload)
		}
	})
	require.Less(t, result.AllocedBytesPerOp(), int64(1024))
}

func TestRequestLatencyBufferedJSONDoesNotFlush(t *testing.T) {
	for _, stringWrite := range []bool{false, true} {
		t.Run(fmt.Sprint(stringWrite), func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			_, clock := requestlatency.New(context.Background())
			writer := &latencyWriter{ResponseWriter: c.Writer, observer: requestlatency.Observer{Timing: clock, Client: true}}
			writer.Header().Set("Content-Type", "application/json")
			payload := `{"data":[{"b64_json":"ok"}]}`
			var err error
			if stringWrite {
				_, err = writer.WriteString(payload)
			} else {
				_, err = writer.Write([]byte(payload))
			}
			require.NoError(t, err)
			require.False(t, rec.Flushed, "instrumentation must preserve buffered JSON transport behavior")
			require.NotNil(t, clock.Snapshot().ClientFirstContentMs)
		})
	}
}

type failedLatencyWriter struct{ gin.ResponseWriter }

func (w *failedLatencyWriter) Write(p []byte) (int, error)       { return len(p), io.ErrClosedPipe }
func (w *failedLatencyWriter) WriteString(s string) (int, error) { return len(s), io.ErrClosedPipe }

func TestRequestLatencyFailedWritesDoNotMarkContent(t *testing.T) {
	for _, stringWrite := range []bool{false, true} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		_, clock := requestlatency.New(context.Background())
		w := &latencyWriter{ResponseWriter: &failedLatencyWriter{c.Writer}, observer: requestlatency.Observer{Timing: clock, Client: true}}
		payload := `{"data":[{"b64_json":"ok"}]}`
		if stringWrite {
			_, _ = w.WriteString(payload)
		} else {
			_, _ = w.Write([]byte(payload))
		}
		w.Flush()
		require.Nil(t, clock.Snapshot().ClientFirstContentMs)
	}
}
