package cmd

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestQueryProjectsReadOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "project spaces#mode=rw%", ".todos")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "issues.db")
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`CREATE TABLE issues (id TEXT, title TEXT, status TEXT, priority TEXT, type TEXT, created_at TEXT, deleted_at TEXT);
		INSERT INTO issues VALUES
		('open-first', 'First', 'open', 'P0', 'task', '2026-01-01', NULL),
		('in-progress', 'Working', 'in_progress', 'P1', 'bug', '2026-01-03', NULL),
		('open-second', 'Second', 'open', 'P1', 'task', '2026-01-02', NULL),
		('closed', 'Finished', 'closed', 'P2', 'task', '2026-01-01', NULL),
		('deleted', 'Removed', 'open', 'P0', 'task', '2026-01-04', '2026-01-05');`)
	closeErr := database.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	before, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	open := []issueRow{
		{ID: "open-first", Title: "First", Status: "open", Priority: "P0", Type: "task"},
		{ID: "in-progress", Title: "Working", Status: "in_progress", Priority: "P1", Type: "bug"},
		{ID: "open-second", Title: "Second", Status: "open", Priority: "P1", Type: "task"},
	}
	for _, tc := range []struct {
		name    string
		showAll bool
		want    []issueRow
	}{
		{name: "open tasks", want: open},
		{name: "including closed", showAll: true, want: append(append([]issueRow{}, open...), issueRow{ID: "closed", Title: "Finished", Status: "closed", Priority: "P2", Type: "task"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := queryIssues(dbPath, tc.showAll)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("issues = %#v, want %#v", got, tc.want)
			}
		})
	}
	after, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("project aggregation changed the database")
	}
}

func TestQueryProjectsDoesNotCreateMissingDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing #database.db")
	if _, err := queryIssues(dbPath, true); err == nil {
		t.Fatal("missing database should fail")
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("query created a missing database: %v", err)
	}
}
