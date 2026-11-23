package rconfig

import (
	"net"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/herraristotle/rad/pkg/model"
)

// Match reports whether req passes cfg. Empty allow lists impose no
// restriction; any matching disallow entry rejects the request.
func (cfg FilterConfig) Match(req *model.Request) bool {
	if req == nil || req.URL == nil {
		return true
	}

	hostname := req.URL.Hostname()
	port := req.URL.Port()
	if port == "" {
		if req.URL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	if !matchHostname(cfg.HostnameAllowed, hostname, true) {
		return false
	}
	if matchHostname(cfg.HostnameDisallowed, hostname, false) {
		return false
	}
	if !matchPort(cfg.PortAllowed, port, true) {
		return false
	}
	if matchPort(cfg.PortDisallowed, port, false) {
		return false
	}
	if !matchPatterns(cfg.PathAllowed, req.URL.Path, true) {
		return false
	}
	if matchPatterns(cfg.PathDisallowed, req.URL.Path, false) {
		return false
	}

	query := req.URL.QueryMap()
	if !matchKeys(cfg.QueryKeyAllowed, query, true) {
		return false
	}
	if matchKeys(cfg.QueryKeyDisallowed, query, false) {
		return false
	}

	if !matchPatterns(cfg.FragmentAllowed, req.URL.Fragment, true) {
		return false
	}
	if matchPatterns(cfg.FragmentDisallowed, req.URL.Fragment, false) {
		return false
	}

	post := req.PostDataMap()
	if !matchKeys(cfg.PostKeyAllowed, post, true) {
		return false
	}
	if matchKeys(cfg.PostKeyDisallowed, post, false) {
		return false
	}
	return true
}

// matchPatterns applies glob patterns to value. When allow is true an empty
// pattern list means "allow everything"; otherwise it means "match nothing".
func matchPatterns(patterns []string, value string, allow bool) bool {
	if len(patterns) == 0 {
		return allow
	}
	for _, pattern := range patterns {
		if globMatch(pattern, value) {
			return true
		}
	}
	return false
}

func matchKeys(patterns []string, values map[string]interface{}, allow bool) bool {
	if len(patterns) == 0 {
		return allow
	}
	for key := range values {
		if matchPatterns(patterns, key, allow) {
			return true
		}
	}
	return false
}

// globMatch reports whether value matches a shell-style pattern. Unlike
// path.Match, "*" is allowed to cross "/" so patterns such as "*admin*"
// work against a whole path.
func globMatch(pattern, value string) bool {
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	re, err := globToRegexp(pattern)
	if err != nil {
		return false
	}
	if re.MatchString(value) {
		return true
	}
	// A pattern without a wildcard or slash also matches a single path
	// segment, so "test" matches "/a/test/b".
	if !strings.ContainsAny(pattern, "*?[") && !strings.Contains(pattern, "/") {
		for _, segment := range strings.Split(value, "/") {
			if segment == pattern {
				return true
			}
		}
	}
	return false
}

// globToRegexp translates *, ? and character classes into a regexp where
// every wildcard may span path separators.
func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("(?s)^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '[':
			end := strings.IndexByte(pattern[i:], ']')
			if end == -1 {
				b.WriteString(regexp.QuoteMeta(string(c)))
				continue
			}
			b.WriteString(pattern[i : i+end+1])
			i += end
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// matchHostname understands t.com, *.t.com, 1.1.1.1, 1.1.1.1/24 and
// ranges such as 1.1-4.1.1-8.
func matchHostname(patterns []string, hostname string, allow bool) bool {
	if len(patterns) == 0 {
		return allow
	}
	for _, pattern := range patterns {
		if hostnameMatch(pattern, hostname) {
			return true
		}
	}
	return false
}

func hostnameMatch(pattern, hostname string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	// CIDR
	if strings.Contains(pattern, "/") {
		if _, network, err := net.ParseCIDR(pattern); err == nil {
			if ip := net.ParseIP(hostname); ip != nil {
				return network.Contains(ip)
			}
		}
		return false
	}
	// IP range, e.g. 1.1-4.1.1-8
	if strings.Contains(pattern, "-") && net.ParseIP(strings.ReplaceAll(pattern, "-", ".")) == nil {
		if inIPRange(pattern, hostname) {
			return true
		}
	}
	if strings.HasPrefix(pattern, "*.") {
		return strings.HasSuffix(hostname, pattern[1:])
	}
	if ok, err := path.Match(pattern, hostname); err == nil && ok {
		return true
	}
	return strings.EqualFold(pattern, hostname)
}

// inIPRange matches an IPv4 address against a "a-b.c.d-e" style range.
func inIPRange(pattern, hostname string) bool {
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	parts := strings.Split(pattern, ".")
	if len(parts) != 4 {
		return false
	}
	for i, part := range parts {
		lo, hi, ok := parseRangePart(part)
		if !ok {
			return false
		}
		if int(ip4[i]) < lo || int(ip4[i]) > hi {
			return false
		}
	}
	return true
}

func parseRangePart(part string) (int, int, bool) {
	if strings.Contains(part, "-") {
		bounds := strings.SplitN(part, "-", 2)
		lo, err1 := strconv.Atoi(bounds[0])
		hi, err2 := strconv.Atoi(bounds[1])
		if err1 != nil || err2 != nil {
			return 0, 0, false
		}
		return lo, hi, true
	}
	n, err := strconv.Atoi(part)
	if err != nil {
		return 0, 0, false
	}
	return n, n, true
}

// matchPort understands 80 and 80-85.
func matchPort(patterns []string, port string, allow bool) bool {
	if len(patterns) == 0 {
		return allow
	}
	value, err := strconv.Atoi(port)
	if err != nil {
		return !allow
	}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if strings.Contains(pattern, "-") {
			bounds := strings.SplitN(pattern, "-", 2)
			lo, err1 := strconv.Atoi(bounds[0])
			hi, err2 := strconv.Atoi(bounds[1])
			if err1 == nil && err2 == nil && value >= lo && value <= hi {
				return true
			}
			continue
		}
		if n, err := strconv.Atoi(pattern); err == nil && n == value {
			return true
		}
	}
	return false
}
