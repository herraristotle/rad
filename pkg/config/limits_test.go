package config

import (
	"testing"
	"time"
)

// TestMaxTabsCeiling pins the documented concurrency ceiling so a future change
// to chaitin/rad parity is a deliberate edit rather than silent drift.
func TestMaxTabsCeiling(t *testing.T) {
	if MaxTabsCeiling != 10 {
		t.Fatalf("MaxTabsCeiling = %d, want 10", MaxTabsCeiling)
	}
	if MaxTabsCount > MaxTabsCeiling {
		t.Fatalf("MaxTabsCount %d exceeds MaxTabsCeiling %d", MaxTabsCount, MaxTabsCeiling)
	}
}

// TestDefaultPageAnalyzeTimeout pins the documented 300s analysis budget that
// applies when page_analyze_timeout_second is omitted.
func TestDefaultPageAnalyzeTimeout(t *testing.T) {
	if DefaultPageAnalyzeTimeout != 300*time.Second {
		t.Fatalf("DefaultPageAnalyzeTimeout = %v, want 300s", DefaultPageAnalyzeTimeout)
	}
}
