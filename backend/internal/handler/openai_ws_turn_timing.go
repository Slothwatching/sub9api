package handler

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/requestlatency"
)

// Admission and completion hooks can run on different relay goroutines. Each
// turn owns a clock; publishing a new turn never resets another turn or ingress.
type openAIWSTurnTimings struct {
	mu     sync.Mutex
	clocks map[int]*requestlatency.Timing
	latest *requestlatency.Timing
}

func (s *openAIWSTurnTimings) begin(turn int, start time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clocks == nil {
		s.clocks = make(map[int]*requestlatency.Timing)
	}
	if s.clocks[turn] == nil {
		_, s.clocks[turn] = requestlatency.ForkAt(context.Background(), start)
		s.latest = s.clocks[turn]
	}
}
func (s *openAIWSTurnTimings) context(parent context.Context, turn int) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	clock := s.clocks[turn]
	if clock == nil {
		// Defensive error/completion path before an admission callback.
		_, clock = requestlatency.ForkAt(parent, time.Now())
		if s.clocks == nil {
			s.clocks = make(map[int]*requestlatency.Timing)
		}
		s.clocks[turn] = clock
		s.latest = clock
	}
	return clock.Context(parent)
}
func (s *openAIWSTurnTimings) finish(turn int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clocks, turn)
}

// Each proxy invocation numbers its first frame as turn 1, even when that frame
// retries a later logical turn. Carry its clock across account/proxy retries.
func (s *openAIWSTurnTimings) seed(clock *requestlatency.Timing) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clocks == nil {
		s.clocks = make(map[int]*requestlatency.Timing)
	}
	s.clocks[1] = clock
	s.latest = clock
}
func (s *openAIWSTurnTimings) current() *requestlatency.Timing {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest
}
