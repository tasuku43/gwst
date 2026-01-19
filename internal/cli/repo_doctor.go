package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/core/output"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/ops/doctor"
	"github.com/tasuku43/gwst/internal/ui"
)

func runDoctor(ctx context.Context, rootDir string, args []string) error {
	doctorFlags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	var fix bool
	var self bool
	var helpFlag bool
	doctorFlags.SetOutput(os.Stdout)
	doctorFlags.Usage = func() {
		printDoctorHelp(os.Stdout)
	}
	doctorFlags.BoolVar(&fix, "fix", false, "remove stale locks only")
	doctorFlags.BoolVar(&self, "self", false, "run self-diagnostics")
	doctorFlags.BoolVar(&helpFlag, "help", false, "show help")
	doctorFlags.BoolVar(&helpFlag, "h", false, "show help")
	if err := doctorFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printDoctorHelp(os.Stdout)
		return nil
	}
	if fix && self {
		return fmt.Errorf("usage: gwst doctor [--fix | --self]")
	}
	if doctorFlags.NArg() != 0 {
		return fmt.Errorf("usage: gwst doctor [--fix | --self]")
	}
	now := time.Now().UTC()
	if self {
		result, err := doctor.SelfCheck(ctx)
		if err != nil {
			return err
		}
		writeDoctorSelfText(result)
		return nil
	}
	if fix {
		result, err := doctor.Fix(ctx, rootDir, now)
		if err != nil {
			return err
		}
		writeDoctorText(result.Result, result.Fixed)
		return nil
	}

	result, err := doctor.Check(ctx, rootDir, now)
	if err != nil {
		return err
	}
	writeDoctorText(result, nil)
	return nil
}

func runRepo(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printRepoHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "get":
		return runRepoGet(ctx, rootDir, args[1:])
	case "ls":
		return runRepoList(ctx, rootDir, args[1:])
	case "rm":
		return runRepoRemove(ctx, rootDir, args[1:], noPrompt)
	default:
		return fmt.Errorf("unknown repo subcommand: %s", args[0])
	}
}

func runRepoGet(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 0 || (len(args) == 1 && isHelpArg(args[0])) {
		printRepoGetHelp(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: gwst repo get <repo>")
	}
	repoSpec := strings.TrimSpace(args[0])
	if repoSpec == "" {
		return fmt.Errorf("repo is required")
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	startSteps(renderer)
	output.Step(formatStep("repo get", displayRepoSpec(repoSpec), repoDestForSpec(rootDir, repoSpec)))

	store, err := repo.Get(ctx, rootDir, repoSpec)
	if err != nil {
		return err
	}
	renderer.Blank()
	renderer.Section("Result")
	renderer.Bullet(fmt.Sprintf("%s %s", store.RepoKey, store.StorePath))
	renderSuggestions(renderer, useColor, []string{
		"gwst create",
		"gwst create --repo <repo>",
	})
	return nil
}

func runRepoList(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printRepoLsHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst repo ls")
	}
	entries, warnings, err := repo.List(rootDir)
	if err != nil {
		return err
	}
	writeRepoListText(entries, warnings)
	return nil
}

type repoRemoveTarget struct {
	Spec      repo.Spec
	SpecInput string
	StorePath string
}

func runRepoRemove(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	rmFlags := flag.NewFlagSet("repo rm", flag.ContinueOnError)
	var helpFlag bool
	rmFlags.BoolVar(&helpFlag, "help", false, "show help")
	rmFlags.BoolVar(&helpFlag, "h", false, "show help")
	rmFlags.SetOutput(os.Stdout)
	rmFlags.Usage = func() {
		printRepoRmHelp(os.Stdout)
	}
	if err := rmFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printRepoRmHelp(os.Stdout)
		return nil
	}

	var repoSpecs []string
	showInputs := true
	if rmFlags.NArg() > 0 {
		repoSpecs = uniqueStringsPreserve(rmFlags.Args())
	} else {
		showInputs = false
		if noPrompt {
			return fmt.Errorf("repo is required with --no-prompt")
		}
		entries, _, err := repo.List(rootDir)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return fmt.Errorf("no repos found")
		}
		var choices []ui.PromptChoice
		for _, entry := range entries {
			label := displayRepoKey(entry.RepoKey)
			value := repoSpecFromKey(entry.RepoKey)
			choices = append(choices, ui.PromptChoice{Label: label, Value: value})
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		selected, err := ui.PromptMultiSelect("gwst repo rm", "repo", choices, theme, useColor)
		if err != nil {
			return err
		}
		repoSpecs = uniqueStringsPreserve(selected)
	}

	if len(repoSpecs) == 0 {
		return fmt.Errorf("at least one repo is required")
	}

	targets, err := resolveRepoRemoveTargets(rootDir, repoSpecs)
	if err != nil {
		return err
	}

	refs, err := findRepoReferences(ctx, rootDir, targets)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		return formatRepoReferenceError(targets, refs)
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)
	if showInputs {
		renderer.Section("Inputs")
		renderer.Bullet("repos")
		var inputs []string
		for _, target := range targets {
			inputs = append(inputs, displayRepoSpec(target.SpecInput))
		}
		renderTreeLines(renderer, inputs, treeLineNormal)
		renderer.Blank()
	}
	renderer.Section("Steps")
	for i, target := range targets {
		output.Step(formatStepWithIndex("remove repo", displayRepoSpec(target.SpecInput), relPath(rootDir, target.StorePath), i+1, len(targets)))
		if err := os.RemoveAll(target.StorePath); err != nil {
			return err
		}
	}
	renderer.Blank()
	renderer.Section("Result")
	for _, target := range targets {
		renderer.Bullet(fmt.Sprintf("%s removed", target.Spec.RepoKey))
	}
	return nil
}

func resolveRepoRemoveTargets(rootDir string, repoSpecs []string) ([]repoRemoveTarget, error) {
	seen := make(map[string]struct{})
	var targets []repoRemoveTarget
	for _, repoSpec := range repoSpecs {
		spec, _, err := repo.Normalize(repoSpec)
		if err != nil {
			return nil, err
		}
		storePath, exists, err := repo.Exists(rootDir, repoSpec)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("repo store not found, run: gwst repo get %s", repoSpec)
		}
		if _, ok := seen[spec.RepoKey]; ok {
			continue
		}
		seen[spec.RepoKey] = struct{}{}
		targets = append(targets, repoRemoveTarget{
			Spec:      spec,
			SpecInput: repoSpec,
			StorePath: storePath,
		})
	}
	return targets, nil
}

func findRepoReferences(ctx context.Context, rootDir string, targets []repoRemoveTarget) (map[string][]string, error) {
	if len(targets) == 0 {
		return nil, nil
	}
	targetKeys := make(map[string]struct{})
	for _, target := range targets {
		targetKeys[target.Spec.RepoKey] = struct{}{}
	}

	entries, _, err := workspace.List(rootDir)
	if err != nil {
		return nil, err
	}
	refs := make(map[string][]string)
	for _, entry := range entries {
		repos, warnings, err := workspace.ScanRepos(ctx, entry.WorkspacePath)
		if err != nil {
			return nil, fmt.Errorf("scan workspace %s: %w", entry.WorkspaceID, err)
		}
		for _, warning := range warnings {
			if warning == nil {
				continue
			}
			if strings.Contains(warning.Error(), "not a git repo") {
				continue
			}
			return nil, fmt.Errorf("workspace %s: %w", entry.WorkspaceID, warning)
		}
		for _, repoEntry := range repos {
			if strings.TrimSpace(repoEntry.RepoKey) == "" {
				label := strings.TrimSpace(repoEntry.Alias)
				if label == "" {
					label = filepath.Base(repoEntry.WorktreePath)
				}
				return nil, fmt.Errorf("workspace %s repo %s has no valid origin", entry.WorkspaceID, label)
			}
			if _, ok := targetKeys[repoEntry.RepoKey]; ok {
				if !containsString(refs[repoEntry.RepoKey], entry.WorkspaceID) {
					refs[repoEntry.RepoKey] = append(refs[repoEntry.RepoKey], entry.WorkspaceID)
				}
			}
		}
	}
	return refs, nil
}

func formatRepoReferenceError(targets []repoRemoveTarget, refs map[string][]string) error {
	var lines []string
	for _, target := range targets {
		usedBy := refs[target.Spec.RepoKey]
		if len(usedBy) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", target.Spec.RepoKey, strings.Join(usedBy, ", ")))
	}
	if len(lines) == 0 {
		return nil
	}
	return fmt.Errorf("repo is used by workspaces:\n%s", strings.Join(lines, "\n"))
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
