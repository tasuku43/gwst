package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/tasuku43/gws/internal/config"
	"github.com/tasuku43/gws/internal/paths"
	"github.com/tasuku43/gws/internal/repo"
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
	case "repo":
		return runRepo(ctx, rootDir, jsonFlag, args[1:])
	case "new":
		return runWorkspaceNew(ctx, rootDir, args[1:])
	case "add":
		return runWorkspaceAdd(ctx, rootDir, args[1:])
	case "ls":
		return runWorkspaceList(ctx, rootDir, jsonFlag, args[1:])
	case "status":
		return runWorkspaceStatus(ctx, rootDir, jsonFlag, args[1:])
	case "rm":
		return runWorkspaceRemove(ctx, rootDir, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runRepo(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("repo subcommand is required")
	}
	switch args[0] {
	case "get":
		return runRepoGet(ctx, rootDir, args[1:])
	case "ls":
		return runRepoList(ctx, rootDir, jsonFlag, args[1:])
	default:
		return fmt.Errorf("unknown repo subcommand: %s", args[0])
	}
}

func runRepoGet(ctx context.Context, rootDir string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gws repo get <repo>")
	}
	store, err := repo.Get(ctx, rootDir, args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\t%s\n", store.RepoKey, store.StorePath)
	return nil
}

func runRepoList(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: gws repo ls")
	}
	entries, warnings, err := repo.List(rootDir)
	if err != nil {
		return err
	}
	if jsonFlag {
		return writeRepoListJSON(entries, warnings)
	}
	writeRepoListText(entries, warnings)
	return nil
}

func runWorkspaceNew(ctx context.Context, rootDir string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gws new <WORKSPACE_ID>")
	}
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	wsDir, err := workspace.New(ctx, rootDir, args[0], cfg)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, wsDir)
	return nil
}

func runWorkspaceAdd(ctx context.Context, rootDir string, args []string) error {
	addFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	var alias string
	addFlags.StringVar(&alias, "alias", "", "worktree alias")
	if err := addFlags.Parse(args); err != nil {
		return err
	}
	if addFlags.NArg() != 2 {
		return fmt.Errorf("usage: gws add <WORKSPACE_ID> <repo> --alias <name>")
	}
	workspaceID := addFlags.Arg(0)
	repoSpec := addFlags.Arg(1)
	if alias == "" {
		return fmt.Errorf("alias is required")
	}
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	repoEntry, err := workspace.Add(ctx, rootDir, workspaceID, repoSpec, alias, cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\t%s\n", repoEntry.Alias, repoEntry.WorktreePath)
	return nil
}

func runWorkspaceList(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: gws ls")
	}
	entries, warnings, err := workspace.List(rootDir)
	if err != nil {
		return err
	}
	if jsonFlag {
		return writeWorkspaceListJSON(entries, warnings)
	}
	writeWorkspaceListText(entries, warnings)
	return nil
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

func runWorkspaceRemove(ctx context.Context, rootDir string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gws rm <WORKSPACE_ID>")
	}
	return workspace.Remove(ctx, rootDir, args[0])
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

type workspaceListJSON struct {
	SchemaVersion int                      `json:"schema_version"`
	Command       string                   `json:"command"`
	Workspaces    []workspaceListEntryJSON `json:"workspaces"`
}

type workspaceListEntryJSON struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspacePath string `json:"workspace_path"`
	ManifestPath  string `json:"manifest_path"`
	RepoCount     int    `json:"repo_count"`
	Warning       string `json:"warning,omitempty"`
}

func writeWorkspaceListJSON(entries []workspace.Entry, warnings []error) error {
	out := workspaceListJSON{
		SchemaVersion: 1,
		Command:       "ls",
	}
	for _, entry := range entries {
		repoCount := 0
		if entry.Manifest != nil {
			repoCount = len(entry.Manifest.Repos)
		}
		item := workspaceListEntryJSON{
			WorkspaceID:   entry.WorkspaceID,
			WorkspacePath: entry.WorkspacePath,
			ManifestPath:  entry.ManifestPath,
			RepoCount:     repoCount,
		}
		if entry.Warning != nil {
			item.Warning = entry.Warning.Error()
		}
		out.Workspaces = append(out.Workspaces, item)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
}

func writeWorkspaceListText(entries []workspace.Entry, warnings []error) {
	fmt.Fprintln(os.Stdout, "id\tpath\trepos")
	for _, entry := range entries {
		repoCount := 0
		if entry.Manifest != nil {
			repoCount = len(entry.Manifest.Repos)
		}
		fmt.Fprintf(os.Stdout, "%s\t%s\t%d\n", entry.WorkspaceID, entry.WorkspacePath, repoCount)
		if entry.Warning != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", entry.WorkspaceID, entry.Warning)
		}
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
}

type repoListJSON struct {
	SchemaVersion int                 `json:"schema_version"`
	Command       string              `json:"command"`
	Repos         []repoListEntryJSON `json:"repos"`
}

type repoListEntryJSON struct {
	RepoKey   string `json:"repo_key"`
	StorePath string `json:"store_path"`
	Warning   string `json:"warning,omitempty"`
}

func writeRepoListJSON(entries []repo.Entry, warnings []error) error {
	out := repoListJSON{
		SchemaVersion: 1,
		Command:       "repo.ls",
	}
	for _, entry := range entries {
		item := repoListEntryJSON{
			RepoKey:   entry.RepoKey,
			StorePath: entry.StorePath,
		}
		out.Repos = append(out.Repos, item)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
}

func writeRepoListText(entries []repo.Entry, warnings []error) {
	fmt.Fprintln(os.Stdout, "repo_key\tstore_path")
	for _, entry := range entries {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", entry.RepoKey, entry.StorePath)
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
}
