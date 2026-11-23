package pkg

import (
	"testing"
	"time"

	"github.com/herraristotle/rad/pkg/config"
)

// TestNormalizeLimitsClampsTabs checks that a config asking for more concurrent
// tabs than chaitin/rad documents is brought down to the ceiling.
func TestNormalizeLimitsClampsTabs(t *testing.T) {
	conf := TaskConfig{MaxTabsCount: 50}
	normalizeLimits(&conf)
	if conf.MaxTabsCount != config.MaxTabsCeiling {
		t.Fatalf("MaxTabsCount = %d, want %d", conf.MaxTabsCount, config.MaxTabsCeiling)
	}
}

// TestNormalizeLimitsKeepsValidTabs checks that a value under the ceiling is
// left alone.
func TestNormalizeLimitsKeepsValidTabs(t *testing.T) {
	conf := TaskConfig{MaxTabsCount: 4}
	normalizeLimits(&conf)
	if conf.MaxTabsCount != 4 {
		t.Fatalf("MaxTabsCount = %d, want 4", conf.MaxTabsCount)
	}
}

// TestNormalizeLimitsDefaultsZeroTabs checks that an unset concurrency falls
// back to the engine default instead of creating a zero-sized pool.
func TestNormalizeLimitsDefaultsZeroTabs(t *testing.T) {
	conf := TaskConfig{}
	normalizeLimits(&conf)
	if conf.MaxTabsCount != config.MaxTabsCount {
		t.Fatalf("MaxTabsCount = %d, want %d", conf.MaxTabsCount, config.MaxTabsCount)
	}
}

// TestNormalizeLimitsBoundsAnalysis checks that an omitted page analysis budget
// becomes the documented default rather than staying unbounded.
func TestNormalizeLimitsBoundsAnalysis(t *testing.T) {
	conf := TaskConfig{}
	normalizeLimits(&conf)
	if conf.PageAnalyzeTimeout != config.DefaultPageAnalyzeTimeout {
		t.Fatalf("PageAnalyzeTimeout = %v, want %v",
			conf.PageAnalyzeTimeout, config.DefaultPageAnalyzeTimeout)
	}
}

// TestNormalizeLimitsKeepsExplicitAnalysis checks that an explicit budget is
// honoured.
func TestNormalizeLimitsKeepsExplicitAnalysis(t *testing.T) {
	conf := TaskConfig{PageAnalyzeTimeout: 45 * time.Second}
	normalizeLimits(&conf)
	if conf.PageAnalyzeTimeout != 45*time.Second {
		t.Fatalf("PageAnalyzeTimeout = %v, want 45s", conf.PageAnalyzeTimeout)
	}
}
