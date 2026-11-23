package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRemoveProfileDirDeletes verifies the owner of a temporary profile
// removes it, including when Chrome has recreated files inside it.
func TestRemoveProfileDirDeletes(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "rad-chrome-test")
	if err := os.MkdirAll(filepath.Join(dir, "Default", "Cache"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Default", "Cache", "data"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	bro := &Browser{profileDir: dir}
	bro.removeProfileDir()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("profile dir still exists after removeProfileDir: %v", err)
	}
	if bro.profileDir != "" {
		t.Fatalf("profileDir not cleared: %q", bro.profileDir)
	}
}

// TestRemoveProfileDirIsNoopWithoutDir guards the early return used by
// ConnectBrowser, which never owns a profile.
func TestRemoveProfileDirIsNoopWithoutDir(t *testing.T) {
	bro := &Browser{}
	bro.removeProfileDir()
	if bro.profileDir != "" {
		t.Fatalf("profileDir should stay empty, got %q", bro.profileDir)
	}
}

// TestBrowserOwnsProfileDir checks that a fresh browser records the profile
// directory it passed to chromedp, so Close is able to clean it up.
func TestBrowserOwnsProfileDir(t *testing.T) {
	if testing.Short() {
		t.Skip("launches a real browser")
	}

	bro := InitBrowserWithOptions(BrowserOptions{ChromiumPath: "/usr/bin/google-chrome"})
	dir := bro.profileDir
	if dir == "" {
		t.Fatal("browser did not record a profile directory")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("profile directory %q not present: %v", dir, err)
	}

	bro.Close()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("profile directory %q survived Close", dir)
	}
}
