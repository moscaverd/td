package syncconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirOverride(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "separate config")
	t.Setenv("TD_CONFIG_DIR", dir)
	got, err := ConfigDir()
	if err != nil || got != dir {
		t.Fatalf("ConfigDir() = %q, %v; want %q", got, err, dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("configuration directory was not created: %v", err)
	}
	if err := SaveAuth(&AuthCredentials{APIKey: "fixture-only"}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dir, "auth.json")); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("auth permissions changed: %v", err)
	}
}

func TestConfigDirRejectsRelativeOverride(t *testing.T) {
	t.Setenv("TD_CONFIG_DIR", "relative-config")
	if _, err := ConfigDir(); err == nil {
		t.Fatal("relative configuration override must fail")
	}
}
