package middleware

import (
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/requestlatency"
	"github.com/gin-gonic/gin"
)

// WithRequestLatency is the first gateway route middleware, before body/auth.
// Only HTTP OpenAI content is observed; WebSocket frames remain unmeasured.
func WithRequestLatency(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if requestlatency.From(c.Request.Context()) != nil {
			next(c)
			return
		}
		ctx, t := requestlatency.New(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		w := &latencyWriter{ResponseWriter: c.Writer, observer: requestlatency.Observer{Timing: t, Client: true}}
		c.Writer = w
		next(c)
	}
}

type latencyWriter struct {
	mu sync.Mutex
	gin.ResponseWriter
	observer       requestlatency.Observer
	contentWritten bool
}

func (w *latencyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.ResponseWriter.Write(p)
	if n > 0 && err == nil && w.Status() >= 200 && w.Status() < 300 {
		w.observer.SSE = strings.Contains(w.Header().Get("Content-Type"), "text/event-stream")
		w.observer.Feed(p[:n])
		w.contentWritten = w.observer.Visible
		// Complete buffered JSON is already rendered. Flush it before taking the
		// usage snapshot so the client timing has the same meaning as streaming.
		if w.contentWritten && !w.observer.SSE && err == nil {
			w.flush()
		}
	}
	return n, err
}
func (w *latencyWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *latencyWriter) Flush()                            { w.mu.Lock(); defer w.mu.Unlock(); w.flush() }
func (w *latencyWriter) flush() {
	w.ResponseWriter.Flush()
	if w.contentWritten {
		w.observer.Timing.Mark(requestlatency.ClientFirstContent)
	}
}
