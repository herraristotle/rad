package rconfig

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestDefaultConfigSendsNoHeaders guards against the default config injecting
// headers into every request. chaitin/rad's generated file has an empty
// domain_headers entry, so rad must not send anything it was not asked to.
func TestDefaultConfigSendsNoHeaders(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(DefaultYAML), &cfg); err != nil {
		t.Fatalf("default config does not parse: %v", err)
	}

	for _, entry := range cfg.DomainHeaders {
		if len(entry.Headers) != 0 {
			t.Fatalf("default domain_headers for %q sets %v; it must be empty",
				entry.Domain, entry.Headers)
		}
	}

	headers := cfg.HeadersFor("example.com", nil)
	if len(headers) != 0 {
		t.Fatalf("default config produced headers %v for a bare crawl", headers)
	}
}

// TestDefaultConfigMatchesChaitinKeys pins the key set of the generated file so
// accidental drift from chaitin/rad's schema is caught.
func TestDefaultConfigMatchesChaitinKeys(t *testing.T) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(DefaultYAML), &raw); err != nil {
		t.Fatalf("default config does not parse: %v", err)
	}

	want := []string{
		"exec_path", "disable_headless", "subdomain", "force_sandbox",
		"enable_image", "parent_path_detect", "proxy", "user_agent",
		"domain_headers", "max_depth", "navigate_timeout_second",
		"load_timeout_second", "retry", "page_analyze_timeout_second",
		"max_interactive", "max_interactive_depth", "max_page_concurrent",
		"max_page_visit", "max_page_visit_per_site", "element_filter_strength",
		"new_task_filter_config", "request_send_filter_config",
		"request_output_filter_config", "entrance_retry", "max_similar_request",
	}
	for _, key := range want {
		if _, ok := raw[key]; !ok {
			t.Errorf("default config is missing key %q", key)
		}
	}
	if len(raw) != len(want) {
		t.Errorf("default config has %d top-level keys, want %d", len(raw), len(want))
	}
}
