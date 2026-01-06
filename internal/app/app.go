package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/tasuku43/gws/internal/config"
	"github.com/tasuku43/gws/internal/doctor"
	"github.com/tasuku43/gws/internal/gc"
	"github.com/tasuku43/gws/internal/initcmd"
	"github.com/tasuku43/gws/internal/paths"
	"github.com/tasuku43/gws/internal/repo"
	"github.com/tasuku43/gws/internal/template"
	"github.com/tasuku43/gws/internal/workspace"
	texttmpl "text/template"
)

// Run is a placeholder for the CLI entrypoint.
func Run() error {
	fs := flag.NewFlagSet("gws", flag.ContinueOnError)
	var rootFlag string
	var jsonFlag bool
	var noPrompt bool
	fs.StringVar(&rootFlag, "root", "", "override gws root")
	fs.BoolVar(&jsonFlag, "json", false, "machine readable output")
	fs.BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompt")
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
	case "init":
		return runInit(rootDir, jsonFlag, args[1:])
	case "doctor":
		return runDoctor(ctx, rootDir, jsonFlag, args[1:])
	case "gc":
		return runGC(ctx, rootDir, jsonFlag, args[1:])
	case "repo":
		return runRepo(ctx, rootDir, jsonFlag, args[1:])
	case "template":
		return runTemplate(ctx, rootDir, jsonFlag, args[1:])
	case "new":
		return runWorkspaceNew(ctx, rootDir, args[1:], noPrompt)
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

func runInit(rootDir string, jsonFlag bool, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: gws init")
	}
	result, err := initcmd.Run(rootDir)
	if err != nil {
		return err
	}
	if jsonFlag {
		return writeInitJSON(result)
	}
	writeInitText(result)
	return nil
}

func runTemplate(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("template subcommand is required")
	}
	switch args[0] {
	case "ls":
		return runTemplateList(ctx, rootDir, jsonFlag, args[1:])
	default:
		return fmt.Errorf("unknown template subcommand: %s", args[0])
	}
}

func runTemplateList(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: gws template ls")
	}
	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}
	names := template.Names(file)
	if jsonFlag {
		return writeTemplateListJSON(names)
	}
	writeTemplateListText(names)
	return nil
}

func runDoctor(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	doctorFlags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	var fix bool
	doctorFlags.BoolVar(&fix, "fix", false, "remove stale locks only")
	if err := doctorFlags.Parse(args); err != nil {
		return err
	}
	if doctorFlags.NArg() != 0 {
		return fmt.Errorf("usage: gws doctor [--fix]")
	}
	now := time.Now().UTC()
	if fix {
		result, err := doctor.Fix(ctx, rootDir, now)
		if err != nil {
			return err
		}
		if jsonFlag {
			return writeDoctorJSON(result.Result, result.Fixed)
		}
		writeDoctorText(result.Result, result.Fixed)
		return nil
	}

	result, err := doctor.Check(ctx, rootDir, now)
	if err != nil {
		return err
	}
	if jsonFlag {
		return writeDoctorJSON(result, nil)
	}
	writeDoctorText(result, nil)
	return nil
}

func runGC(ctx context.Context, rootDir string, jsonFlag bool, args []string) error {
	gcFlags := flag.NewFlagSet("gc", flag.ContinueOnError)
	var dryRun bool
	var older string
	gcFlags.BoolVar(&dryRun, "dry-run", false, "only list candidates")
	gcFlags.StringVar(&older, "older", "", "older than duration (e.g. 30d, 720h)")
	if err := gcFlags.Parse(args); err != nil {
		return err
	}
	if gcFlags.NArg() != 0 {
		return fmt.Errorf("usage: gws gc [--dry-run] [--older <duration>]")
	}

	olderThan, err := parseOlder(older)
	if err != nil {
		return err
	}

	opts := gc.Options{OlderThan: olderThan}
	now := time.Now().UTC()
	if dryRun {
		result, err := gc.DryRun(ctx, rootDir, opts, now)
		if err != nil {
			return err
		}
		if jsonFlag {
			return writeGCJSON(result, true, older)
		}
		writeGCText(result, true, older)
		return nil
	}

	result, err := gc.Run(ctx, rootDir, opts, now)
	if err != nil {
		return err
	}
	if jsonFlag {
		return writeGCJSON(result, false, older)
	}
	writeGCText(result, false, older)
	return nil
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

func runWorkspaceNew(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	newFlags := flag.NewFlagSet("new", flag.ContinueOnError)
	var templateName string
	newFlags.StringVar(&templateName, "template", "", "template name")
	if err := newFlags.Parse(args); err != nil {
		return err
	}
	if newFlags.NArg() > 1 {
		return fmt.Errorf("usage: gws new [--template <name>] [<WORKSPACE_ID>]")
	}

	workspaceID := ""
	if newFlags.NArg() == 1 {
		workspaceID = newFlags.Arg(0)
	}

	if templateName == "" || workspaceID == "" {
		if noPrompt {
			return fmt.Errorf("template name and workspace id are required without prompt")
		}
		var err error
		templateName, workspaceID, err = promptTemplateAndID(rootDir, templateName, workspaceID)
		if err != nil {
			return err
		}
	}

	cfg, err := config.Load(rootDir)
	if err != nil {
		return err
	}

	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}
	tmpl, ok := file.Templates[templateName]
	if !ok {
		return fmt.Errorf("template not found: %s", templateName)
	}
	if err := preflightTemplateRepos(ctx, rootDir, tmpl); err != nil {
		return err
	}

	wsDir, err := workspace.New(ctx, rootDir, workspaceID, cfg)
	if err != nil {
		return err
	}

	if err := applyTemplate(ctx, rootDir, workspaceID, tmpl, cfg); err != nil {
		if rollbackErr := workspace.Remove(ctx, rootDir, workspaceID); rollbackErr != nil {
			return fmt.Errorf("apply template failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	fmt.Fprintln(os.Stdout, wsDir)
	return nil
}

func runWorkspaceAdd(ctx context.Context, rootDir string, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: gws add <WORKSPACE_ID> <repo>")
	}
	workspaceID := args[0]
	repoSpec := args[1]
	cfg, err := config.Load(rootDir)
	if err != nil {
		return err
	}
	repoEntry, err := workspace.Add(ctx, rootDir, workspaceID, repoSpec, "", cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\t%s\n", repoEntry.Alias, repoEntry.WorktreePath)
	return nil
}

func promptTemplateAndID(rootDir, templateName, workspaceID string) (string, string, error) {
	file, err := template.Load(rootDir)
	if err != nil {
		return "", "", err
	}
	names := template.Names(file)
	if len(names) == 0 {
		return "", "", fmt.Errorf("no templates found in %s", filepath.Join(rootDir, template.FileName))
	}

	if templateName == "" {
		selected, err := promptSelect("template", names)
		if err != nil {
			return "", "", err
		}
		templateName = selected
	}

	if workspaceID == "" {
		input, err := promptText("workspace id", true)
		if err != nil {
			return "", "", err
		}
		workspaceID = input
	}

	return templateName, workspaceID, nil
}

func promptText(label string, required bool) (string, error) {
	validate := func(input string) error {
		if required && strings.TrimSpace(input) == "" {
			return fmt.Errorf("required")
		}
		return nil
	}
	funcMap := cloneFuncMap(promptui.FuncMap)
	prompt := promptui.Prompt{
		Label:    label,
		Validate: validate,
		Templates: &promptui.PromptTemplates{
			Prompt:  "{{ . }}: ",
			Valid:   "{{ . }}: ",
			Invalid: "{{ . }}: ",
			Success: "{{ . }}",
			FuncMap: funcMap,
		},
	}
	value, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptSelect(label string, items []string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select")
	}
	displayItem := func(value string) string {
		if value == "done" {
			return value
		}
		return strings.TrimSuffix(value, ".git")
	}
	funcMap := cloneFuncMap(promptui.FuncMap)
	funcMap["trimGit"] = func(item string) string {
		return displayItem(item)
	}
	sel := promptui.Select{
		Label:             label,
		Items:             items,
		Size:              min(10, len(items)),
		HideHelp:          false,
		StartInSearchMode: true,
		Searcher: func(input string, index int) bool {
			item := displayItem(items[index])
			input = strings.ToLower(strings.TrimSpace(input))
			item = strings.ToLower(item)
			return strings.Contains(item, input)
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "> {{ . | trimGit }}",
			Inactive: "  {{ . | trimGit }}",
			Selected: "{{ . | trimGit }}",
			FuncMap:  funcMap,
		},
	}
	_, result, err := sel.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}

func cloneFuncMap(src texttmpl.FuncMap) texttmpl.FuncMap {
	dest := texttmpl.FuncMap{}
	for key, value := range src {
		dest[key] = value
	}
	return dest
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func preflightTemplateRepos(ctx context.Context, rootDir string, tmpl template.Template) error {
	var missing []string
	for _, repoSpec := range tmpl.Repos {
		if strings.TrimSpace(repoSpec) == "" {
			return fmt.Errorf("template repo is empty")
		}
		if _, err := repo.Open(ctx, rootDir, repoSpec); err != nil {
			missing = append(missing, repoSpec)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("repo get required for: %s", strings.Join(missing, ", "))
	}
	return nil
}

func applyTemplate(ctx context.Context, rootDir, workspaceID string, tmpl template.Template, cfg config.Config) error {
	for _, repoSpec := range tmpl.Repos {
		if _, err := workspace.Add(ctx, rootDir, workspaceID, repoSpec, "", cfg); err != nil {
			return err
		}
	}
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

type templateListJSON struct {
	SchemaVersion int      `json:"schema_version"`
	Command       string   `json:"command"`
	Templates     []string `json:"templates"`
}

type templateShowJSON struct {
	SchemaVersion int      `json:"schema_version"`
	Command       string   `json:"command"`
	Name          string   `json:"name"`
	Repos         []string `json:"repos"`
}

type templateRemoveJSON struct {
	SchemaVersion int    `json:"schema_version"`
	Command       string `json:"command"`
	Name          string `json:"name"`
	Removed       bool   `json:"removed"`
}

func writeTemplateListJSON(names []string) error {
	out := templateListJSON{
		SchemaVersion: 1,
		Command:       "template.ls",
		Templates:     names,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeTemplateListText(names []string) {
	for _, name := range names {
		fmt.Fprintln(os.Stdout, name)
	}
}

func writeTemplateShowJSON(name string, tmpl template.Template) error {
	out := templateShowJSON{
		SchemaVersion: 1,
		Command:       "template.show",
		Name:          name,
		Repos:         tmpl.Repos,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeTemplateShowText(name string, tmpl template.Template) {
	fmt.Fprintf(os.Stdout, "%s\n", name)
	for _, repo := range tmpl.Repos {
		fmt.Fprintf(os.Stdout, " - %s\n", repo)
	}
}

func writeTemplateRemoveJSON(name string) error {
	out := templateRemoveJSON{
		SchemaVersion: 1,
		Command:       "template.rm",
		Name:          name,
		Removed:       true,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

type initJSON struct {
	SchemaVersion int      `json:"schema_version"`
	Command       string   `json:"command"`
	RootDir       string   `json:"root_dir"`
	CreatedDirs   []string `json:"created_dirs"`
	CreatedFiles  []string `json:"created_files"`
	SkippedDirs   []string `json:"skipped_dirs"`
	SkippedFiles  []string `json:"skipped_files"`
}

func writeInitJSON(result initcmd.Result) error {
	out := initJSON{
		SchemaVersion: 1,
		Command:       "init",
		RootDir:       result.RootDir,
		CreatedDirs:   result.CreatedDirs,
		CreatedFiles:  result.CreatedFiles,
		SkippedDirs:   result.SkippedDirs,
		SkippedFiles:  result.SkippedFiles,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeInitText(result initcmd.Result) {
	fmt.Fprintf(os.Stdout, "root: %s\n", result.RootDir)
	for _, dir := range result.CreatedDirs {
		fmt.Fprintf(os.Stdout, "created\t%s\n", dir)
	}
	for _, file := range result.CreatedFiles {
		fmt.Fprintf(os.Stdout, "created\t%s\n", file)
	}
	for _, dir := range result.SkippedDirs {
		fmt.Fprintf(os.Stdout, "exists\t%s\n", dir)
	}
	for _, file := range result.SkippedFiles {
		fmt.Fprintf(os.Stdout, "exists\t%s\n", file)
	}
}

type gcJSON struct {
	SchemaVersion int               `json:"schema_version"`
	Command       string            `json:"command"`
	DryRun        bool              `json:"dry_run"`
	Older         string            `json:"older,omitempty"`
	Candidates    []gcCandidateJSON `json:"candidates"`
}

type gcCandidateJSON struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspacePath string `json:"workspace_path"`
	LastUsedAt    string `json:"last_used_at"`
	Reason        string `json:"reason"`
}

func writeGCJSON(result gc.Result, dryRun bool, older string) error {
	out := gcJSON{
		SchemaVersion: 1,
		Command:       "gc",
		DryRun:        dryRun,
		Older:         strings.TrimSpace(older),
	}
	for _, candidate := range result.Candidates {
		out.Candidates = append(out.Candidates, gcCandidateJSON{
			WorkspaceID:   candidate.WorkspaceID,
			WorkspacePath: candidate.WorkspacePath,
			LastUsedAt:    candidate.LastUsedAt,
			Reason:        candidate.Reason,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
}

func writeGCText(result gc.Result, dryRun bool, older string) {
	action := "gc"
	if dryRun {
		action = "gc --dry-run"
	}
	if strings.TrimSpace(older) != "" {
		fmt.Fprintf(os.Stdout, "%s (older=%s)\n", action, older)
	}
	fmt.Fprintln(os.Stdout, "id\tlast_used_at\treason\tpath")
	for _, candidate := range result.Candidates {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%s\n", candidate.WorkspaceID, candidate.LastUsedAt, candidate.Reason, candidate.WorkspacePath)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
}

func parseOlder(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	if strings.HasSuffix(trimmed, "d") {
		raw := strings.TrimSuffix(trimmed, "d")
		days, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("invalid --older value: %s", value)
		}
		if days < 0 {
			return 0, fmt.Errorf("invalid --older value: %s", value)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	parsed, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid --older value: %s", value)
	}
	return parsed, nil
}

type doctorJSON struct {
	SchemaVersion int               `json:"schema_version"`
	Command       string            `json:"command"`
	Issues        []doctorIssueJSON `json:"issues"`
	Fixed         []string          `json:"fixed,omitempty"`
}

type doctorIssueJSON struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

func writeDoctorJSON(result doctor.Result, fixed []string) error {
	out := doctorJSON{
		SchemaVersion: 1,
		Command:       "doctor",
	}
	for _, issue := range result.Issues {
		out.Issues = append(out.Issues, doctorIssueJSON{
			Kind:    issue.Kind,
			Path:    issue.Path,
			Message: issue.Message,
		})
	}
	if len(fixed) > 0 {
		out.Fixed = fixed
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	return nil
}

func writeDoctorText(result doctor.Result, fixed []string) {
	fmt.Fprintln(os.Stdout, "kind\tpath\tmessage")
	for _, issue := range result.Issues {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", issue.Kind, issue.Path, issue.Message)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warning)
	}
	if len(fixed) > 0 {
		for _, path := range fixed {
			fmt.Fprintf(os.Stdout, "fixed\t%s\n", path)
		}
	}
}
