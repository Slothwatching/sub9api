package requestlatency

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimingObserverOversizedLineSplitRecovery(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n"} {
		for _, splitEnding := range []bool{false, true} {
			for _, bufferedOverflow := range []bool{false, true} {
				line := "data: " + strings.Repeat("x", maxObservation)
				if bufferedOverflow {
					if ending != "\r\n" {
						continue
					}
					// Exactly full: the following CR overflows the line buffer.
					line = line[:maxObservation]
				}
				for _, split := range []int{0, 5, len(line) / 2, len(line) - 1, len(line)} {
					t.Run(fmt.Sprintf("ending=%q/splitEnding=%t/bufferedOverflow=%t/split=%d", ending, splitEnding, bufferedOverflow, split), func(t *testing.T) {
						_, clock := New(context.Background())
						o := Observer{Timing: clock, SSE: true}
						o.FeedString("event: response.output_text.delta" + ending)
						o.FeedString(line[:split])
						o.FeedString(line[split:])
						if splitEnding {
							o.FeedString(strings.TrimSuffix(ending, "\n"))
							o.FeedString("\n")
						} else {
							o.FeedString(ending)
						}
						require.True(t, o.dropping, "terminating the oversized line is not a blank delimiter")
						o.FeedString(`data: {"delta":"same oversized event"}` + ending)
						require.True(t, o.dropping)
						o.End()
						require.Equal(t, Snapshot{}, clock.Snapshot(), "End must not observe dropped data")
						// Split the blank CRLF delimiter too; CR alone must not open a dropped line.
						o.FeedString(strings.TrimSuffix(ending, "\n"))
						require.True(t, o.dropping)
						o.FeedString("\n")
						require.False(t, o.dropping)
						require.Empty(t, o.event)
						o.FeedString(`data: {"delta":"untyped"}` + ending + ending)
						require.Nil(t, clock.Snapshot().FirstVisibleContentMs)
						o.FeedString("event: image_generation.completed" + ending + `data: {"b64_json":"later"}` + ending + ending)
						require.True(t, o.Visible)
						require.NotNil(t, clock.Snapshot().FirstVisibleContentMs)
						require.LessOrEqual(t, len(o.buf), maxObservation)
						require.LessOrEqual(t, len(o.data), maxObservation)
					})
				}
			}
		}
	}
}

func TestTimingObserverOversizedStreamChunkBoundaries(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n"} {
		for _, continuation := range []bool{false, true} {
			prefix := "event: response.output_text.delta" + ending
			line := "data: " + strings.Repeat("x", maxObservation+1)
			dropped := prefix + line + ending
			if continuation {
				dropped += `data: {"delta":"must not observe"}` + ending
			}
			delimiterEnd := len(dropped) + len(ending)
			stream := dropped + ending + "event: response.output_text.delta" + ending + `data: {"delta":"later"}` + ending + ending
			// Every boundary near fields and delimiters, plus representative payload splits.
			splits := []int{0, len(prefix) + 6, len(prefix) + len(line)/2}
			for i := len(prefix) + len(line) - 2; i <= len(stream); i++ {
				splits = append(splits, i)
			}
			for _, split := range splits {
				t.Run(fmt.Sprintf("ending=%q/continuation=%t/split=%d", ending, continuation, split), func(t *testing.T) {
					_, clock := New(context.Background())
					o := Observer{Timing: clock, SSE: true}
					check := func(consumed int) {
						if consumed >= len(prefix)+len(line) && consumed < delimiterEnd {
							require.True(t, o.dropping)
						}
						if consumed >= delimiterEnd {
							require.False(t, o.dropping)
						}
						if consumed < len(stream) {
							require.False(t, o.Visible)
							require.Equal(t, Snapshot{}, clock.Snapshot())
						}
						require.LessOrEqual(t, len(o.buf), maxObservation)
						require.LessOrEqual(t, len(o.data), maxObservation)
					}
					o.FeedString(stream[:split])
					check(split)
					// Byte-sized remaining chunks exercise CR/LF and field fragmentation.
					for i := split; i < len(stream); i++ {
						o.FeedString(stream[i : i+1])
						check(i + 1)
					}
					require.True(t, o.Visible)
					require.NotNil(t, clock.Snapshot().FirstVisibleContentMs)
				})
			}
		}
	}
}
