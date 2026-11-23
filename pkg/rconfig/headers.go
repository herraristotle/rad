package rconfig

import (
	"strings"

	"github.com/herraristotle/rad/pkg/model"
)

// HeadersFor returns the extra headers that apply to req: the global headers
// first, then every domain_headers entry whose glob matches the hostname. Later
// entries win, which lets a specific domain override a "*" entry.
func (cfg *Config) HeadersFor(hostname string, global map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for key, value := range global {
		result[key] = value
	}
	for _, entry := range cfg.DomainHeaders {
		if !globMatch(entry.Domain, hostname) {
			continue
		}
		for key, value := range entry.Headers {
			result[key] = value
		}
	}
	if cfg.UserAgent != "" {
		result["User-Agent"] = cfg.UserAgent
	}
	return result
}

// ApplyHeaders merges the resolved headers into req, without touching an
// existing value when neither the config nor the globals provide one.
func (cfg *Config) ApplyHeaders(req *model.Request, global map[string]interface{}) {
	if req == nil || req.URL == nil {
		return
	}
	headers := cfg.HeadersFor(req.URL.Hostname(), global)
	if len(headers) == 0 {
		return
	}
	if req.Headers == nil {
		req.Headers = map[string]interface{}{}
	}
	for key, value := range headers {
		req.Headers[key] = value
	}
}

// DomainForRequest is a small helper used by callers that only have a raw URL.
func (cfg *Config) DomainForRequest(rawURL string) string {
	if idx := strings.Index(rawURL, "://"); idx >= 0 {
		rest := rawURL[idx+3:]
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			rest = rest[:slash]
		}
		if colon := strings.IndexByte(rest, ':'); colon >= 0 {
			rest = rest[:colon]
		}
		return rest
	}
	return rawURL
}

// DomainMatch reports whether hostname matches a domain_headers glob.
func DomainMatch(pattern, hostname string) bool {
	return globMatch(pattern, hostname)
}
