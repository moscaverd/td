package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterLoadAndPrune(t *testing.T) {
	t.Setenv("TD_CONFIG_DIR", filepath.Join(t.TempDir(), "config"))
	project := filepath.Join(t.TempDir(), "project spaces#percent%")
	if err := os.MkdirAll(filepath.Join(project, ".todos"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".todos", "issues.db"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if added, err := Register(project); err != nil || !added {
		t.Fatalf("register = %v, %v", added, err)
	}
	if added, err := Register(project); err != nil || added {
		t.Fatalf("duplicate register = %v, %v", added, err)
	}
	entries, err := Load()
	if err != nil || len(entries) != 1 {
		t.Fatalf("load = %#v, %v", entries, err)
	}
	if entries[0].Path != project || entries[0].Name != filepath.Base(project) || entries[0].RegisteredAt.IsZero() {
		t.Fatalf("unexpected registry entry: %#v", entries[0])
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := Register(missing); err != nil {
		t.Fatal(err)
	}
	valid, removed, err := Prune()
	if err != nil || removed != 1 || len(valid) != 1 || valid[0].Path != project {
		t.Fatalf("prune = %#v, %d, %v", valid, removed, err)
	}
	loaded, err := Load()
	if err != nil || len(loaded) != 1 || loaded[0].Path != project {
		t.Fatalf("persisted prune = %#v, %v", loaded, err)
	}
	if data, err := os.ReadFile(filepath.Join(project, ".todos", "issues.db")); err != nil || string(data) != "fixture" {
		t.Fatalf("registry changed database bytes: %q, %v", data, err)
	}
}
