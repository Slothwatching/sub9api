// Package requestlatency implements fork-local, ingress-based gateway timings.
// It deliberately does not redefine legacy first_token_ms or scheduler feedback.
package requestlatency

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"sync"
	"time"
)

type key struct{}

// Snapshot contains nullable milliseconds; absent observations stay nil.
// Event timestamps are offsets from ingress; queue/connect/TLS/attempt are durations.
type Snapshot struct {
	UserQueueMs           *int `json:"user_queue_ms"`
	AccountQueueMs        *int `json:"account_queue_ms"`
	AccountSelectionMs    *int `json:"account_selection_ms"`
	UpstreamConnectMs     *int `json:"upstream_connect_ms"`
	UpstreamTLSMs         *int `json:"upstream_tls_ms"`
	UpstreamFirstByteMs   *int `json:"upstream_first_byte_ms"`
	FirstDataMs           *int `json:"first_data_ms"`
	FirstSemanticMs       *int `json:"first_semantic_ms"`
	FirstVisibleContentMs *int `json:"first_visible_content_ms"`
	ClientFirstContentMs  *int `json:"client_first_content_ms"`
	UserDurationMs        *int `json:"user_duration_ms"`
	AttemptCount          *int `json:"attempt_count"`
	LastAttemptMs         *int `json:"last_attempt_ms"`
	LastFailedAttemptMs   *int `json:"last_failed_attempt_ms"`
}

type Metric uint8

const (
	UserQueue Metric = iota
	AccountQueue
	AccountSelection
	UpstreamConnect
	UpstreamTLS
	UpstreamFirstByte
	FirstData
	FirstSemantic
	FirstVisibleContent
	ClientFirstContent
	UserDuration
	AttemptCount
	LastAttempt
	LastFailedAttempt
	metricCount
)

type Timing struct {
	mu       sync.Mutex
	start    time.Time
	values   [metricCount]time.Duration
	seen     uint32
	attempts int
}

func New(ctx context.Context) (context.Context, *Timing) {
	if t := From(ctx); t != nil {
		return ctx, t
	}
	t := &Timing{start: time.Now()}
	return context.WithValue(ctx, key{}, t), t
}
func From(ctx context.Context) *Timing {
	if ctx == nil {
		return nil
	}
	t, _ := ctx.Value(key{}).(*Timing)
	return t
}
func (t *Timing) Mark(m Metric) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.seen&(1<<m) == 0 {
		t.values[m] = time.Since(t.start)
		t.seen |= 1 << m
	}
}
func (t *Timing) Add(m Metric, d time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.values[m] += d
	t.seen |= 1 << m
}

// Segment includes both immediate acquisition and waits, summed across failover.
func (t *Timing) Segment(m Metric) func() {
	if t == nil {
		return func() {}
	}
	start := time.Now()
	return func() { t.Add(m, time.Since(start)) }
}
func (t *Timing) BeginAttempt() func(bool) {
	if t == nil {
		return func(bool) {}
	}
	start := time.Now()
	t.mu.Lock()
	t.attempts++
	t.seen |= 1 << AttemptCount
	t.mu.Unlock()
	return func(failed bool) {
		t.mu.Lock()
		defer t.mu.Unlock()
		d := time.Since(start)
		t.values[LastAttempt] = d
		t.seen |= 1 << LastAttempt
		if failed {
			t.values[LastFailedAttempt] = d
			t.seen |= 1 << LastFailedAttempt
		}
	}
}

// Snapshot is detached from the live clock, safe to send to an async usage worker.
func (t *Timing) Snapshot() Snapshot {
	if t == nil {
		return Snapshot{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	var v [metricCount]int
	var p [metricCount]*int
	for i := Metric(0); i < metricCount; i++ {
		if t.seen&(1<<i) != 0 {
			v[i] = int(t.values[i].Milliseconds())
			p[i] = &v[i]
		}
	}
	if p[AttemptCount] != nil {
		v[AttemptCount] = t.attempts
	}
	return Snapshot{
		UserQueueMs:           p[UserQueue],
		AccountQueueMs:        p[AccountQueue],
		AccountSelectionMs:    p[AccountSelection],
		UpstreamConnectMs:     p[UpstreamConnect],
		UpstreamTLSMs:         p[UpstreamTLS],
		UpstreamFirstByteMs:   p[UpstreamFirstByte],
		FirstDataMs:           p[FirstData],
		FirstSemanticMs:       p[FirstSemantic],
		FirstVisibleContentMs: p[FirstVisibleContent],
		ClientFirstContentMs:  p[ClientFirstContent],
		UserDurationMs:        p[UserDuration],
		AttemptCount:          p[AttemptCount],
		LastAttemptMs:         p[LastAttempt],
		LastFailedAttemptMs:   p[LastFailedAttempt],
	}
}

// Trace composes with existing traces on the actual outbound request. GotConn
// includes pool wait, DNS, TCP, proxy CONNECT and TLS; TLS is also split out.
// Unsupported/plugin transports leave connect/TLS nil rather than guessing.
func (t *Timing) Trace(ctx context.Context) context.Context {
	if t == nil {
		return ctx
	}
	var mu sync.Mutex
	var connStart, tlsStart time.Time
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GetConn: func(string) { mu.Lock(); connStart = time.Now(); mu.Unlock() },
		GotConn: func(httptrace.GotConnInfo) {
			mu.Lock()
			s := connStart
			mu.Unlock()
			if !s.IsZero() {
				t.Add(UpstreamConnect, time.Since(s))
			}
		},
		TLSHandshakeStart: func() { mu.Lock(); tlsStart = time.Now(); mu.Unlock() },
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			mu.Lock()
			s := tlsStart
			mu.Unlock()
			if !s.IsZero() {
				t.Add(UpstreamTLS, time.Since(s))
			}
		},
		GotFirstResponseByte: func() { t.Mark(UpstreamFirstByte) },
	})
}
