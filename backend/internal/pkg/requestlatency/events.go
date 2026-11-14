package requestlatency

import (
	"bytes"
	"io"
	"strings"
	"unsafe"

	"github.com/tidwall/gjson"
)

// Classify separates structural/semantic progress from usable content for both
// Responses and Chat. Role-only, usage-only and lifecycle events never qualify.
func Classify(data []byte) (semantic, visible bool) { return classify(data, "") }
func classify(data []byte, event string) (semantic, visible bool) {
	if !gjson.ValidBytes(data) {
		return false, false
	}
	r := rawJSON(strings.TrimSpace(byteString(data)))
	// Only the small discriminator is decoded; payload strings stay raw.
	typ := ""
	if v := r.get("type"); len(v) <= 256 {
		typ = gjson.Parse(string(v)).Str
	}
	if typ == "" {
		typ = event
	}
	if choices := r.get("choices"); choices.array() {
		choices.each(func(choice rawJSON) {
			for _, part := range []string{"delta", "message"} {
				d := choice.get(part)
				for _, field := range []string{"content", "reasoning_content", "reasoning", "refusal", "audio.data", "audio.transcript", "function_call.arguments"} {
					v := d.get(field)
					visible = visible || v.nonempty()
					if field == "content" {
						v.each(func(part rawJSON) { visible = visible || itemVisible(part) })
					}
				}
				semantic = semantic || d.get("function_call.name").value()
				d.get("tool_calls").each(func(tool rawJSON) {
					semantic = semantic || tool.get("id").value() || tool.get("function.name").value()
					visible = visible || tool.get("function.arguments").value()
				})
			}
		})
		return semantic || visible, visible
	}
	switch typ {
	case "response.created", "response.queued", "response.in_progress", "response.completed", "response.done", "response.failed", "response.incomplete", "error", "ping", "keepalive", "metadata", "response.metadata", "codex.rate_limits", "codex.response.metadata", "response.usage", "response.usage.updated", "usage":
		// Terminal envelopes can carry complete content, but lifecycle alone is skipped.
		if typ != "response.completed" && typ != "response.done" {
			return false, false
		}
	}
	if strings.HasPrefix(typ, "response.") {
		if strings.HasSuffix(typ, ".delta") {
			visible = r.get("delta").value()
		}
		for _, path := range []string{"text", "arguments", "input", "refusal", "partial_image_b64"} {
			if r.get(path).value() {
				visible = true
			}
		}
		if itemVisible(r.get("item")) || itemVisible(r.get("part")) {
			visible = true
		}
		r.get("response.output").each(func(item rawJSON) { visible = visible || itemVisible(item) })
		lifecycle := typ == "response.completed" || typ == "response.done"
		return visible || !lifecycle, visible
	}
	if strings.HasPrefix(typ, "image_generation.") || strings.HasPrefix(typ, "image_edit.") {
		visible = r.get("b64_json").nonempty() || r.get("url").nonempty()
		return visible, visible
	}
	// Buffered Responses and native images (successful JSON only).
	r.get("output").each(func(item rawJSON) { visible = visible || itemVisible(item) })
	r.get("data").each(func(item rawJSON) { visible = visible || item.get("b64_json").nonempty() || item.get("url").nonempty() })
	return visible, visible
}
func itemVisible(r rawJSON) bool {
	for _, key := range []string{"text", "transcript", "arguments", "input", "result", "refusal"} {
		if r.get(key).value() {
			return true
		}
	}
	visible := false
	for _, key := range []string{"content", "summary"} {
		r.get(key).each(func(part rawJSON) { visible = visible || itemVisible(part) })
	}
	return visible
}

// Observer bounds line/event buffers to 64 KiB each and stops after content. Oversized events are
// skipped; instrumentation must not duplicate unbounded image or tool payloads.
// Feed accepts fragmented SSE frames as well as buffered JSON responses.
type Observer struct {
	Timing   *Timing
	Client   bool
	SSE      bool
	event    string
	buf      []byte
	data     []byte
	dropping bool
	done     bool
	Visible  bool
}

const maxObservation = 64 * 1024

func (o *Observer) observe(data []byte) {
	if len(bytes.TrimSpace(data)) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("[DONE]")) {
		return
	}
	semantic, visible := classify(data, o.event)
	if !o.Client {
		o.Timing.Mark(FirstData)
		if semantic {
			o.Timing.Mark(FirstSemantic)
		}
		if visible {
			o.Timing.Mark(FirstVisibleContent)
		}
	}
	if visible {
		o.Visible = true
		o.done = true
		o.buf = nil
		o.data = nil
	}
}

// Stopped reports whether further payload inspection is unnecessary.
func (o *Observer) Stopped() bool { return o.Timing == nil || o.done }

// FeedString observes a read-only view only for this call. No view of s survives.
func (o *Observer) FeedString(s string) {
	o.Feed(unsafe.Slice(unsafe.StringData(s), len(s)))
}
func (o *Observer) Feed(p []byte) {
	if o.Timing == nil || o.done {
		return
	}
	if !o.SSE {
		// A complete buffered JSON write/read already has storage owned by its caller.
		// Inspect it in place, including large images, without a second payload copy.
		if len(o.buf) == 0 && gjson.ValidBytes(p) {
			o.observe(p)
			o.done = true
			return
		}
		if len(o.buf)+len(p) > maxObservation {
			o.done = true
			o.buf = nil
			return
		}
		o.buf = append(o.buf, p...)
		if gjson.ValidBytes(o.buf) {
			o.observe(o.buf)
		}
		return
	}
	for len(p) > 0 && !o.done {
		i := bytes.IndexByte(p, '\n')
		n := len(p)
		if i >= 0 {
			n = i
		}
		if len(o.buf)+n > maxObservation {
			o.dropping = true
			o.buf = nil
			o.data = nil
		}
		if !o.dropping {
			o.buf = append(o.buf, p[:n]...)
		}
		p = p[n:]
		if i < 0 {
			return
		}
		p = p[1:]
		line := bytes.TrimSpace(o.buf)
		if o.dropping {
			if n == 0 {
				o.dropping = false
			}
		} else if len(line) == 0 {
			if len(o.data) > 0 {
				o.observe(o.data)
				o.data = o.data[:0]
			}
			o.event = ""
		} else if bytes.HasPrefix(line, []byte("event:")) {
			o.event = string(bytes.TrimSpace(line[6:]))
		} else if bytes.HasPrefix(line, []byte("data:")) {
			data := bytes.TrimSpace(line[5:])
			if len(o.data)+len(data)+1 > maxObservation {
				o.dropping = true
				o.data = nil
			} else {
				if len(o.data) > 0 {
					o.data = append(o.data, '\n')
				}
				o.data = append(o.data, data...)
			}
		}
		o.buf = o.buf[:0]
	}
}
func (o *Observer) End() {
	if o.done || o.dropping {
		return
	}
	if o.SSE && len(o.buf) > 0 {
		o.Feed([]byte("\n"))
	}
	if len(o.data) > 0 {
		o.observe(o.data)
		o.data = nil
	}
}

type observedBody struct {
	io.ReadCloser
	observer Observer
}

func (b *observedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.observer.Timing.Mark(UpstreamFirstByte)
		b.observer.Feed(p[:n])
	}
	if err == io.EOF {
		b.observer.End()
	}
	return n, err
}
func ObserveBody(body io.ReadCloser, t *Timing, sse bool) io.ReadCloser {
	if body == nil || t == nil {
		return body
	}
	return &observedBody{ReadCloser: body, observer: Observer{Timing: t, SSE: sse}}
}
