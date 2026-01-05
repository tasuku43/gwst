package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/tasuku43/gws/internal/paths"
	"github.com/tasuku43/gws/internal/workspace"
)

// Run is a placeholder for the CLI entrypoint.
func Run() error {
	fs := flag.NewFlagSet("gws", flag.ContinueOnError)
	var rootFlag string
	var jsonFlag bool
	fs.StringVar(&rootFlag, "root", "", "override gws root")
	fs.BoolVar(&jsonFlag, "json", false, "machine readable output")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	args := fs.Args()
	if len(args) == 0 {
		return fmt.Errorf("command is required")
	}

	rootDir, err := paths.ResolveRoot(rootFlag)
	if err != nil {
		return err
	}

	ctx := context.Background()
	switch args[0] {
	case "status":
		return runWorkspaceStatus(ctx, rootDir, jsonFlag, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runWorkspaceStatus(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gws status <WORKSPACE_ID>")
	}
	workspaceID := args[0]
	result, err := workspace.Status(ctx, rootDir, workspaceID)
	if err != nil {
		return err
	}

	if jsonFlag {
		return writeWorkspaceStatusJSON(result)
	}
	writeWorkspaceStatusText(result)
	return nil
}

type workspaceStatusJSON struct {
	SchemaVersion int                       `json:"schema_version"`
	Command       string                    `json:"command"`
	WorkspaceID   string                    `json:"workspace_id"`
	Repos         []workspaceStatusRepoJSON `json:"repos"`
}

type workspaceStatusRepoJSON struct {
	Alias          string `json:"alias"`
	Branch         string `json:"branch"`
	Head           string `json:"head,omitempty"`
	Dirty          bool   `json:"dirty"`
	UntrackedCount int    `json:"untracked_count"`
	Error          string `json:"error,omitempty"`
}

func writeWorkspaceStatusJSON(result workspace.StatusResult) error {
	out := workspaceStatusJSON{
		SchemaVersion: 1,
		Command:       "status",
		WorkspaceID:   result.WorkspaceID,
	}
	for _, repo := range result.Repos {
		repoOut := workspaceStatusRepoJSON{
			Alias:          repo.Alias,
			Branch:         repo.Branch,
			Head:           repo.Head,
			Dirty:          repo.Dirty,
			UntrackedCount: repo.UntrackedCount,
		}
		if repo.Error != nil {
			repoOut.Error = repo.Error.Error()
		}
		out.Repos = append(out.Repos, repoOut)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeWorkspaceStatusText(result workspace.StatusResult) {
	fmt.Fprintln(os.Stdout, "alias\tbranch\thead\tdirty\tuntracked")
	for _, repo := range result.Repos {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%t\t%d\n", repo.Alias, repo.Branch, repo.Head, repo.Dirty, repo.UntrackedCount)
		if repo.Error != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", repo.Alias, repo.Error)
		}
	}
}
