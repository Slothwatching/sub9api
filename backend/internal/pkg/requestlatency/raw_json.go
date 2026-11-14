package requestlatency

import (
	"strings"
	"unsafe"

	"github.com/tidwall/gjson"
)

// byteString is a read-only, call-scoped view. Neither classification nor the
// observer retains it: callers may reuse their read/write buffer after Feed.
func byteString(p []byte) string {
	return unsafe.String(unsafe.SliceData(p), len(p))
}

// rawJSON traverses validated JSON without decoding string values. gjson.Get
// decodes escaped strings even if only Raw is needed, which can allocate an
// entire image/tool value. These views only retain slices during classification.
type rawJSON string

func (r rawJSON) nonempty() bool { return len(r) > 2 && r[0] == '"' }

// value mirrors the old String()!="" check for non-string tool inputs too.
func (r rawJSON) value() bool {
	if len(r) == 0 || r == "null" {
		return false
	}
	if r[0] == '"' {
		return r.nonempty()
	}
	return true
}
func (r rawJSON) array() bool { return len(r) > 0 && r[0] == '[' }

// rawEnd skips one value in validated JSON, including nested arrays/objects.
func rawEnd(s string, i int) int {
	depth := 0
	for i < len(s) {
		switch s[i] {
		case '"':
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			if depth == 0 {
				return i
			}
			continue
		case '{', '[':
			depth++
		case '}', ']':
			if depth == 0 {
				return i
			}
			depth--
			if depth == 0 {
				return i + 1
			}
		case ',', ':', ' ', '\t', '\r', '\n':
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return i
}
func rawSpace(s string, i int) int {
	for i < len(s) && s[i] <= ' ' {
		i++
	}
	return i
}
func (r rawJSON) get(path string) rawJSON {
	key, rest, nested := strings.Cut(path, ".")
	if len(r) == 0 || r[0] != '{' {
		return ""
	}
	s := string(r)
	for i := rawSpace(s, 1); i < len(s) && s[i] == '"'; {
		end := rawEnd(s, i)
		name := s[i:end]
		i = rawSpace(s, rawSpace(s, end)+1) // colon
		end = rawEnd(s, i)
		// Bound even escaped key decoding; a key matching our small fixed names
		// cannot have a raw encoding larger than six bytes per character.
		matches := len(name) == len(key)+2 && name[1:len(name)-1] == key
		if !matches && len(name) <= len(key)*6+2 && strings.Contains(name, "\\") {
			matches = gjson.Parse(name).Str == key
		}
		if matches {
			value := rawJSON(s[i:end])
			if nested {
				return value.get(rest)
			}
			return value
		}
		i = rawSpace(s, end)
		if i >= len(s) || s[i] != ',' {
			break
		}
		i = rawSpace(s, i+1)
	}
	return ""
}
func (r rawJSON) each(visit func(rawJSON)) {
	if !r.array() {
		return
	}
	s := string(r)
	for i := rawSpace(s, 1); i < len(s) && s[i] != ']'; {
		end := rawEnd(s, i)
		visit(rawJSON(s[i:end]))
		i = rawSpace(s, end)
		if i >= len(s) || s[i] != ',' {
			return
		}
		i = rawSpace(s, i+1)
	}
}
