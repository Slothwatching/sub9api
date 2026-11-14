package requestlatency

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func largeImageJSON() []byte {
	// Escapes catch both a copied parse string and decoding/materializing b64_json.
	return []byte(`{"data":[{"b64_json":"` + strings.Repeat(`A\u0041`, 256*1024) + `"}]}`)
}

func benchmarkCompleteJSON(b *testing.B, payload []byte) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, clock := New(context.Background())
		observer := Observer{Timing: clock}
		observer.Feed(payload)
		if !observer.Visible || cap(observer.buf) != 0 {
			b.Fatal("complete image content not observed directly")
		}
	}
}

func BenchmarkTimingLargeCompleteJSON(b *testing.B) { benchmarkCompleteJSON(b, largeImageJSON()) }

func TestTimingLargeCompleteJSONAllocations(t *testing.T) {
	for name, payload := range map[string][]byte{
		"buffered image":     largeImageJSON(),
		"native image event": []byte(`{"type":"image_generation.completed","b64_json":"` + strings.Repeat(`A\u0041`, 256*1024) + `"}`),
		"tool arguments":     []byte(`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"` + strings.Repeat(`A\u0041`, 256*1024) + `"}}]}}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			result := testing.Benchmark(func(b *testing.B) { benchmarkCompleteJSON(b, payload) })
			require.Less(t, result.AllocedBytesPerOp(), int64(len(payload)/16), "must not allocate a payload-sized parse string or decoded value")
			t.Logf("payload=%d B observation=%d B/op, %d allocs/op", len(payload), result.AllocedBytesPerOp(), result.AllocsPerOp())
		})
	}
}

func TestTimingNativeImageEvents(t *testing.T) {
	for _, prefix := range []string{"image_generation.", "image_edit."} {
		for _, suffix := range []string{"partial_image", "completed", "future_content"} {
			for _, field := range []string{"b64_json", "url"} {
				for _, value := range []string{`"ok"`, `""`, `null`} {
					payload := []byte(`{"type":"` + prefix + suffix + `","` + field + `":` + value + `}`)
					semantic, visible := Classify(payload)
					require.Equal(t, value == `"ok"`, visible, string(payload))
					require.Equal(t, visible, semantic)
					// Event-name fallback is used when type is omitted from an SSE payload.
					_, clock := New(context.Background())
					observer := Observer{Timing: clock, SSE: true}
					observer.Feed([]byte("event: " + prefix + suffix + "\ndata: {\"" + field + "\":" + value + "}\n\n"))
					require.Equal(t, visible, observer.Visible)
				}
			}
		}
		for _, suffix := range []string{"in_progress", "completed", "error"} {
			_, visible := Classify([]byte(`{"type":"` + prefix + suffix + `"}`))
			require.False(t, visible)
		}
	}
	for _, payload := range []string{`{"type":"error"}`, `{"type":"ping"}`, `{"type":"image_generation.completed","b64_json":null}`} {
		_, visible := Classify([]byte(payload))
		require.False(t, visible)
	}
	// Raw traversal handles whitespace, nested containers and escaped field names.
	_, visible := Classify([]byte(` { "data": [ {"nested":{"x":["}","\\\""]},"b64_\u006ason":"\u0041"} ] } `))
	require.True(t, visible)
}
