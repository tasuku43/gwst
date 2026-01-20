package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/preset"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/infra/output"
	"github.com/tasuku43/gwst/internal/ui"
)

func runPreset(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printPresetHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "ls":
		return runPresetList(ctx, rootDir, args[1:])
	case "add":
		return runPresetAdd(ctx, rootDir, args[1:], noPrompt)
	case "rm":
		return runPresetRemove(ctx, rootDir, args[1:], noPrompt)
	case "validate":
		return runPresetValidate(ctx, rootDir, args[1:])
	default:
		return fmt.Errorf("unknown preset subcommand: %s", args[0])
	}
}

func runPresetList(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printPresetLsHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst preset ls")
	}
	file, err := preset.Load(rootDir)
	if err != nil {
		return err
	}
	names := preset.Names(file)
	writePresetListText(file, names)
	return nil
}

func uniqueStringsPreserve(items []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func runPresetAdd(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	addFlags := flag.NewFlagSet("preset add", flag.ContinueOnError)
	var helpFlag bool
	var repos stringSliceFlag
	addFlags.Var(&repos, "repo", "repo spec (repeatable)")
	addFlags.BoolVar(&helpFlag, "help", false, "show help")
	addFlags.BoolVar(&helpFlag, "h", false, "show help")
	addFlags.SetOutput(os.Stdout)
	addFlags.Usage = func() {
		printPresetAddHelp(os.Stdout)
	}
	if err := addFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printPresetAddHelp(os.Stdout)
		return nil
	}
	if addFlags.NArg() > 1 {
		return fmt.Errorf("usage: gwst preset add [<name>] [--repo <repo> ...]")
	}

	name := ""
	if addFlags.NArg() == 1 {
		name = addFlags.Arg(0)
	}

	file, err := preset.Load(rootDir)
	if err != nil {
		return err
	}

	repoSpecs := preset.NormalizeRepos(repos)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	if strings.TrimSpace(name) == "" && len(repoSpecs) == 0 {
		if noPrompt {
			return fmt.Errorf("preset name and repos are required with --no-prompt")
		}
		choices, err := buildPresetRepoChoices(rootDir)
		if err != nil {
			return err
		}
		if len(choices) == 0 {
			return fmt.Errorf("no repos found; run gwst repo get first")
		}
		name, repoSpecs, err = ui.PromptPresetRepos("gwst preset add", name, choices, theme, useColor)
		if err != nil {
			return err
		}
		repoSpecs = preset.NormalizeRepos(repoSpecs)
	} else {
		if strings.TrimSpace(name) == "" {
			if noPrompt {
				return fmt.Errorf("preset name is required with --no-prompt")
			}
			name, err = ui.PromptPresetName("gwst preset add", "", theme, useColor)
			if err != nil {
				return err
			}
		}
		if len(repoSpecs) == 0 {
			if noPrompt {
				return fmt.Errorf("repos are required with --no-prompt")
			}
			choices, err := buildPresetRepoChoices(rootDir)
			if err != nil {
				return err
			}
			if len(choices) == 0 {
				return fmt.Errorf("no repos found; run gwst repo get first")
			}
			var selected []string
			name, selected, err = ui.PromptPresetRepos("gwst preset add", name, choices, theme, useColor)
			if err != nil {
				return err
			}
			repoSpecs = preset.NormalizeRepos(selected)
		}
	}

	if err := preset.ValidateName(name); err != nil {
		return err
	}
	if _, exists := file.Presets[name]; exists {
		return fmt.Errorf("preset already exists: %s", name)
	}

	if len(repoSpecs) == 0 {
		return fmt.Errorf("at least one repo is required")
	}

	for _, repoSpec := range repoSpecs {
		if _, _, err := repo.Normalize(repoSpec); err != nil {
			return err
		}
		if _, exists, err := repo.Exists(rootDir, repoSpec); err != nil {
			return err
		} else if !exists {
			return fmt.Errorf("repo store not found, run: gwst repo get %s", repoSpec)
		}
	}

	if file.Presets == nil {
		file.Presets = map[string]preset.Preset{}
	}
	file.Presets[name] = preset.Preset{Repos: repoSpecs}

	if err := preset.Save(rootDir, file); err != nil {
		return err
	}

	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	renderer.Section("Result")
	renderer.Bullet(name)
	var reposDisplay []string
	for _, repoSpec := range repoSpecs {
		reposDisplay = append(reposDisplay, displayPresetRepo(repoSpec))
	}
	renderTreeLines(renderer, reposDisplay, treeLineNormal)
	renderSuggestions(renderer, useColor, []string{
		"gwst create --preset",
		"gwst create --preset <name>",
	})
	return nil
}

func runPresetRemove(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	rmFlags := flag.NewFlagSet("preset rm", flag.ContinueOnError)
	var helpFlag bool
	rmFlags.BoolVar(&helpFlag, "help", false, "show help")
	rmFlags.BoolVar(&helpFlag, "h", false, "show help")
	rmFlags.SetOutput(os.Stdout)
	rmFlags.Usage = func() {
		printPresetRmHelp(os.Stdout)
	}
	if err := rmFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printPresetRmHelp(os.Stdout)
		return nil
	}

	file, err := preset.Load(rootDir)
	if err != nil {
		return err
	}

	var names []string
	showInputs := true
	if rmFlags.NArg() > 0 {
		names = uniqueStringsPreserve(rmFlags.Args())
		for _, name := range names {
			if err := preset.ValidateName(name); err != nil {
				return err
			}
		}
	} else {
		showInputs = false
		if noPrompt {
			return fmt.Errorf("preset name is required with --no-prompt")
		}
		presets := preset.Names(file)
		if len(presets) == 0 {
			return fmt.Errorf("no presets found in %s", filepath.Join(rootDir, manifest.FileName))
		}
		var choices []ui.PromptChoice
		for _, name := range presets {
			choices = append(choices, ui.PromptChoice{Label: name, Value: name})
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		selected, err := ui.PromptMultiSelect("gwst preset rm", "preset", choices, theme, useColor)
		if err != nil {
			return err
		}
		names = uniqueStringsPreserve(selected)
	}

	for _, name := range names {
		if _, exists := file.Presets[name]; !exists {
			return fmt.Errorf("preset not found: %s", name)
		}
	}

	for _, name := range names {
		delete(file.Presets, name)
	}

	if err := preset.Save(rootDir, file); err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)
	if showInputs {
		renderer.Section("Inputs")
		renderer.Bullet("presets")
		renderTreeLines(renderer, names, treeLineNormal)
		renderer.Blank()
	}
	renderer.Section("Steps")
	for i, name := range names {
		output.Step(formatStepWithIndex("remove preset", name, relPath(rootDir, filepath.Join(rootDir, manifest.FileName)), i+1, len(names)))
	}
	renderer.Blank()
	renderer.Section("Result")
	for _, name := range names {
		renderer.Bullet(fmt.Sprintf("%s removed", name))
	}
	return nil
}

func runPresetValidate(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printPresetValidateHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst preset validate")
	}
	result, err := preset.Validate(rootDir)
	if err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	renderer.Section("Result")
	if len(result.Issues) == 0 {
		renderer.Bullet("no issues found")
		return nil
	}

	for _, issue := range result.Issues {
		renderer.BulletError(issue.Kind)
		details := presetIssueDetails(issue, result.Path)
		if len(details) > 0 {
			renderTreeLines(renderer, details, treeLineError)
		}
	}
	return fmt.Errorf("preset validation failed")
}

func buildPresetRepoChoices(rootDir string) ([]ui.PromptChoice, error) {
	repos, _, err := repo.List(rootDir)
	if err != nil {
		return nil, err
	}
	var choices []ui.PromptChoice
	for _, entry := range repos {
		label := displayRepoKey(entry.RepoKey)
		value := repoSpecFromKey(entry.RepoKey)
		choices = append(choices, ui.PromptChoice{Label: label, Value: value})
	}
	return choices, nil
}
