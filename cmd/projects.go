package cmd

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/marcus/td/internal/output"
	"github.com/marcus/td/internal/registry"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"pj"},
	Short:   "List all registered td projects",
	Long:    `Shows all td projects on this machine. Auto-prunes projects whose database no longer exists.`,
	GroupID: "query",
	RunE:    runProjectsList,
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Aggregate open tasks across all projects",
	RunE:  runProjectsAggregate,
}

var projectsCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove stale project entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, removed, err := registry.Prune()
		if err != nil {
			output.Error("prune failed: %v", err)
			return err
		}
		if removed == 0 {
			fmt.Println("No stale entries found.")
		} else {
			output.Success("Removed %d stale project(s)", removed)
		}
		return nil
	},
}

func runProjectsList(cmd *cobra.Command, args []string) error {
	entries, removed, err := registry.Prune()
	if err != nil {
		output.Error("load registry: %v", err)
		return err
	}
	if removed > 0 {
		output.Warning("Pruned %d stale project(s)", removed)
	}

	if len(entries) == 0 {
		fmt.Println("No projects registered. Run 'td init' in a project to register it.")
		return nil
	}

	fmt.Printf("%-25s  %s\n", "PROJECT", "PATH")
	for _, e := range entries {
		fmt.Printf("%-25s  %s\n", e.Name, e.Path)
	}

	return nil
}

// issueRow holds the fields we query from each project's database.
type issueRow struct {
	ID       string
	Title    string
	Status   string
	Priority string
	Type     string
}

func runProjectsAggregate(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")

	entries, removed, err := registry.Prune()
	if err != nil {
		output.Error("load registry: %v", err)
		return err
	}
	if removed > 0 {
		output.Warning("Pruned %d stale project(s)", removed)
	}

	if len(entries) == 0 {
		fmt.Println("No projects registered.")
		return nil
	}

	for _, entry := range entries {
		dbPath := filepath.Join(entry.Path, ".todos", "issues.db")
		issues, err := queryIssues(dbPath, showAll)
		if err != nil {
			output.Warning("%s: %v", entry.Name, err)
			continue
		}

		if len(issues) == 0 {
			continue
		}

		fmt.Printf("\n%s  (%s)\n", strings.ToUpper(entry.Name), entry.Path)
		fmt.Printf("%s\n", strings.Repeat("─", 60))
		for _, iss := range issues {
			fmt.Printf("  %-10s  [%s]  %-13s  %s\n", iss.ID, iss.Priority, iss.Status, iss.Title)
		}
	}

	fmt.Println()
	return nil
}

func queryIssues(dbPath string, showAll bool) ([]issueRow, error) {
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	query := `SELECT id, title, status, priority, type FROM issues WHERE deleted_at IS NULL`
	if !showAll {
		query += ` AND status NOT IN ('closed')`
	}
	query += ` ORDER BY priority ASC, created_at DESC LIMIT 50`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var issues []issueRow
	for rows.Next() {
		var iss issueRow
		if err := rows.Scan(&iss.ID, &iss.Title, &iss.Status, &iss.Priority, &iss.Type); err != nil {
			continue
		}
		issues = append(issues, iss)
	}
	return issues, nil
}

func init() {
	projectsListCmd.Flags().BoolP("all", "a", false, "Include closed tasks")

	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsCleanCmd)
	rootCmd.AddCommand(projectsCmd)
}
