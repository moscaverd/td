package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/marcus/td/internal/syncconfig"
)

// Entry represents a registered td project.
type Entry struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	RegisteredAt time.Time `json:"registered_at"`
}

const registryFile = "projects.json"

// registryPath returns the full path to ~/.config/td/projects.json.
func registryPath() (string, error) {
	dir, err := syncconfig.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, registryFile), nil
}

// Load reads the project registry from disk. Returns empty slice if file doesn't exist.
func Load() ([]Entry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Save writes the project registry to disk.
func Save(entries []Entry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Register adds a project to the registry if not already present.
// Returns true if newly added, false if already registered.
func Register(projectPath string) (bool, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return false, err
	}

	entries, err := Load()
	if err != nil {
		return false, err
	}

	// Check if already registered
	for _, e := range entries {
		if e.Path == absPath {
			return false, nil
		}
	}

	name := filepath.Base(absPath)
	entries = append(entries, Entry{
		Path:         absPath,
		Name:         name,
		RegisteredAt: time.Now().UTC(),
	})

	if err := Save(entries); err != nil {
		return false, err
	}
	return true, nil
}

// Prune removes entries whose .todos/issues.db no longer exists.
// Returns the cleaned list and count of removed entries.
func Prune() ([]Entry, int, error) {
	entries, err := Load()
	if err != nil {
		return nil, 0, err
	}

	var valid []Entry
	for _, e := range entries {
		dbPath := filepath.Join(e.Path, ".todos", "issues.db")
		if _, err := os.Stat(dbPath); err == nil {
			valid = append(valid, e)
		}
	}

	removed := len(entries) - len(valid)
	if removed > 0 {
		if err := Save(valid); err != nil {
			return nil, 0, err
		}
	}

	return valid, removed, nil
}
