package filter

import (
	"net/url"
	"strings"
	"sync"

	"github.com/herraristotle/rad/pkg/model"
)

// ElementFilter removes requests that are structurally identical to an
// already-seen request on the same site. It models chaitin/rad's
// element_filter_strength: 0 disables it, and 1..7 make the fingerprint
// progressively more aggressive, so more "similar" requests collapse into one.
//
// The fingerprint is built from the request's shape rather than its exact
// values: the path with numeric segments normalised, parameter names, and (at
// higher strengths) parameter value classes. Two requests that produce the same
// fingerprint are considered the same element.
type ElementFilter struct {
	// Strength is 0..7. 0 disables filtering entirely.
	Strength int
	// MaxSimilar caps how many requests may share one fingerprint on a site.
	// Zero or negative means unlimited. This is chaitin/rad's
	// max_similar_request.
	MaxSimilar int

	seen sync.Map // fingerprint -> struct{}
}

// NewElementFilter builds a filter. strength is clamped to 0..7.
func NewElementFilter(strength int) *ElementFilter {
	if strength < 0 {
		strength = 0
	}
	if strength > 7 {
		strength = 7
	}
	return &ElementFilter{Strength: strength}
}

// DoFilter reports whether req should be filtered out as a duplicate element.
func (f *ElementFilter) DoFilter(req *model.Request) bool {
	if f == nil || f.Strength <= 0 || req == nil || req.URL == nil {
		return false
	}
	fp := f.fingerprint(req)
	if f.MaxSimilar > 0 {
		// Count how many requests share this shape; drop past the limit.
		actual, _ := f.seen.LoadOrStore(fp, &similarCounter{})
		counter := actual.(*similarCounter)
		if counter.increment() > f.MaxSimilar {
			return true
		}
		return false
	}
	if _, loaded := f.seen.LoadOrStore(fp, struct{}{}); loaded {
		return true
	}
	return false
}

// similarCounter tracks occurrences of one fingerprint.
type similarCounter struct {
	mu sync.Mutex
	n  int
}

func (c *similarCounter) increment() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	return c.n
}

// fingerprint encodes the request's structure at the configured strength.
func (f *ElementFilter) fingerprint(req *model.Request) string {
	var b strings.Builder

	b.WriteString(req.URL.Hostname())
	b.WriteString("|")
	b.WriteString(normalizePathSegments(req.URL.Path, f.Strength))
	b.WriteString("|")

	// Query: always include parameter names; values only as strength rises.
	b.WriteString(queryFingerprint(req.URL.RawQuery, f.Strength))
	b.WriteString("|")

	// Method counts as part of the element's shape.
	b.WriteString(req.Method)

	// At the top strengths, POST bodies are folded in too.
	if f.Strength >= 6 && req.PostData != "" {
		b.WriteString("|")
		b.WriteString(valueClass(req.PostData))
	}
	return b.String()
}

// normalizePathSegments replaces variable-looking path segments with a marker.
// The higher the strength, the more segments are collapsed.
func normalizePathSegments(path string, strength int) string {
	if path == "" || path == "/" {
		return "/"
	}
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, segment := range segments {
		switch {
		case isNumericSegment(segment):
			segments[i] = "{{num}}"
		case strength >= 3 && looksLikeID(segment):
			segments[i] = "{{id}}"
		case strength >= 5 && looksLikeHash(segment):
			segments[i] = "{{hash}}"
		case strength >= 7 && looksLikeDateSegment(segment):
			segments[i] = "{{date}}"
		}
	}
	return "/" + strings.Join(segments, "/")
}

// queryFingerprint encodes query keys, and at higher strengths value classes.
func queryFingerprint(rawQuery string, strength int) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)

	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		if strength >= 4 {
			b.WriteString("=")
			// Use the first value to classify the value's shape.
			class := valueClass(values[key][0])
			b.WriteString(class)
		}
		b.WriteString("&")
	}
	return b.String()
}

// valueClass buckets a value into a coarse shape, mirroring the idea that
// "123", "456" and "789" are the same kind of value.
func valueClass(value string) string {
	switch {
	case value == "":
		return "empty"
	case isNumericSegment(value):
		return "num"
	case looksLikeHash(value):
		return "hash"
	case looksLikeDateSegment(value):
		return "date"
	default:
		return "str"
	}
}

func isNumericSegment(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// looksLikeID matches ids such as 12345, 1a2b3c, or uuid-ish strings.
func looksLikeID(s string) bool {
	if len(s) < 8 {
		return false
	}
	hasDigit := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '-', r == '_':
		default:
			return false
		}
	}
	return hasDigit
}

// looksLikeHash matches long hex-ish tokens.
func looksLikeHash(s string) bool {
	if len(s) < 16 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// looksLikeDateSegment matches 2024, 2024-01, 2024-01-02 and 20240102.
func looksLikeDateSegment(s string) bool {
	digits := strings.ReplaceAll(strings.ReplaceAll(s, "-", ""), "/", "")
	if len(digits) < 4 {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// sortStrings is a tiny insertion sort to avoid pulling in sort for a handful
// of keys; it keeps the fingerprint deterministic.
func sortStrings(items []string) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j] < items[j-1]; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// Reset clears the seen set, used when starting a new crawl.
func (f *ElementFilter) Reset() {
	if f == nil {
		return
	}
	f.seen = sync.Map{}
}
