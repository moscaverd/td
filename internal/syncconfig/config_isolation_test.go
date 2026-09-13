package syncconfig

import (
	"os"
	"testing"
)

// Isolate inherited defaults as well as commands that update the project registry.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "td-test-config-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("TD_CONFIG_DIR", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	os.Exit(code)
}
