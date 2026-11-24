package pkg

import (
	"testing"
	"time"

	"github.com/herraristotle/rad/pkg/config"
)

// The generated rad_config.yml is the documented source of truth for defaults.
// These tests pin the engine constants to those documented values so that
// running with a config file and running with CLI flags alone behave the same.

// TestDefaultConcurrencyMatchesConfig checks max_page_concurrent.
func TestDefaultConcurrencyMatchesConfig(t *testing.T) {
	if config.MaxTabsCount != 10 {
		t.Fatalf("MaxTabsCount = %d, want 10 (max_page_concurrent)", config.MaxTabsCount)
	}
}

// TestDefaultMaxCrawlMatchesConfig checks max_page_visit.
func TestDefaultMaxCrawlMatchesConfig(t *testing.T) {
	if config.MaxCrawlCount != 1000 {
		t.Fatalf("MaxCrawlCount = %d, want 1000 (max_page_visit)", config.MaxCrawlCount)
	}
}

// TestDefaultNavigateTimeoutMatchesConfig checks navigate_timeout_second.
func TestDefaultNavigateTimeoutMatchesConfig(t *testing.T) {
	if config.TabRunTimeout != 10*time.Second {
		t.Fatalf("TabRunTimeout = %v, want 10s (navigate_timeout_second)", config.TabRunTimeout)
	}
}

// TestDefaultLoadTimeoutMatchesConfig checks load_timeout_second.
func TestDefaultLoadTimeoutMatchesConfig(t *testing.T) {
	if config.DomContentLoadedTimeout != 10*time.Second {
		t.Fatalf("DomContentLoadedTimeout = %v, want 10s (load_timeout_second)",
			config.DomContentLoadedTimeout)
	}
}
