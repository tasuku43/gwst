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
	"github.com/tasuku43/gion/internal/domain/manifest"
	"github.com/tasuku43/gion/internal/domain/preset"
	"github.com/tasuku43/gion/internal/domain/repo"
	"github.com/tasuku43/gion/internal/ui"
)

func runManifestPreset(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printManifestPresetHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "ls":
		return runManifestPresetList(ctx, rootDir, args[1:])
	case "add":
		return runManifestPresetAdd(ctx, rootDir, args[1:], noPrompt)
	case "rm":
		return runManifestPresetRemove(ctx, rootDir, args[1:], noPrompt)
	case "validate":
		return runManifestPresetValidate(ctx, rootDir, args[1:])
	default:
		return fmt.Errorf("unknown manifest preset subcommand: %s", args[0])
	}
}

func runManifestPresetList(ctx context.Context, rootDir string, args []string) error {
	lsFlags := flag.NewFlagSet("manifest preset ls", flag.ContinueOnError)
	lsFlags.SetOutput(os.Stdout)
	var helpFlag bool
	var noPrompt bool
	lsFlags.BoolVar(&helpFlag, "help", false, "show help")
	lsFlags.BoolVar(&helpFlag, "h", false, "show help")
	lsFlags.BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompt (no effect)")
	lsFlags.Usage = func() {
		printManifestPresetLsHelp(os.Stdout)
	}
	if err := lsFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = noPrompt
	if helpFlag {
		printManifestPresetLsHelp(os.Stdout)
		return nil
	}
	if lsFlags.NArg() != 0 {
		return fmt.Errorf("usage: gion manifest preset ls [--no-prompt]")
	}

	file, err := preset.Load(rootDir)
	if err != nil {
		return err
	}
	names := preset.Names(file)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)

	if len(names) > 0 {
		renderer.Section("Info")
		renderer.Bullet(fmt.Sprintf("presets: %d", len(names)))
		renderer.Blank()
	}

	renderer.Section("Result")
	if len(names) == 0 {
		renderer.Bullet("no presets found")
		return nil
	}

	for _, name := range names {
		entry, ok := file.Presets[name]
		if !ok {
			continue
		}
		renderer.Bullet(name)
		var reposDisplay []string
		for _, repoSpec := range entry.Repos {
			reposDisplay = append(reposDisplay, displayPresetRepo(repoSpec))
		}
		renderTreeLines(renderer, reposDisplay, treeLineNormal)
	}
	_ = ctx
	return nil
}

func runManifestPresetAdd(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	addFlags := flag.NewFlagSet("manifest preset add", flag.ContinueOnError)
	var helpFlag bool
	var repos stringSliceFlag
	addFlags.Var(&repos, "repo", "repo spec (repeatable)")
	addFlags.BoolVar(&helpFlag, "help", false, "show help")
	addFlags.BoolVar(&helpFlag, "h", false, "show help")
	addFlags.SetOutput(os.Stdout)
	addFlags.Usage = func() {
		printManifestPresetAddHelp(os.Stdout)
	}
	if err := addFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printManifestPresetAddHelp(os.Stdout)
		return nil
	}
	if addFlags.NArg() > 1 {
		return fmt.Errorf("usage: gion manifest preset add [<name>] [--repo <repo> ...] [--no-prompt]")
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
	prompted := false

	if strings.TrimSpace(name) == "" && len(repoSpecs) == 0 {
		if noPrompt {
			return fmt.Errorf("preset name and repos are required with --no-prompt")
		}
		choices, err := buildManifestPresetRepoChoices(rootDir)
		if err != nil {
			return err
		}
		if len(choices) == 0 {
			return fmt.Errorf("no repos found; run gion repo get first")
		}
		name, repoSpecs, err = ui.PromptPresetRepos("gion manifest preset add", name, choices, theme, useColor)
		if err != nil {
			return err
		}
		prompted = true
		repoSpecs = preset.NormalizeRepos(repoSpecs)
	} else {
		if strings.TrimSpace(name) == "" {
			if noPrompt {
				return fmt.Errorf("preset name is required with --no-prompt")
			}
			name, err = ui.PromptPresetName("gion manifest preset add", "", theme, useColor)
			if err != nil {
				return err
			}
			prompted = true
		}
		if len(repoSpecs) == 0 {
			if noPrompt {
				return fmt.Errorf("repos are required with --no-prompt")
			}
			choices, err := buildManifestPresetRepoChoices(rootDir)
			if err != nil {
				return err
			}
			if len(choices) == 0 {
				return fmt.Errorf("no repos found; run gion repo get first")
			}
			var selected []string
			name, selected, err = ui.PromptPresetRepos("gion manifest preset add", name, choices, theme, useColor)
			if err != nil {
				return err
			}
			prompted = true
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
			return fmt.Errorf("repo store not found, run: gion repo get %s", repoSpec)
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
	if !prompted {
		renderer.Section("Inputs")
		renderer.Bullet(fmt.Sprintf("preset name: %s", name))
		renderer.Bullet("repos")
		var reposDisplay []string
		for _, repoSpec := range repoSpecs {
			reposDisplay = append(reposDisplay, displayPresetRepo(repoSpec))
		}
		renderTreeLines(renderer, reposDisplay, treeLineNormal)
		renderer.Blank()
	}
	renderer.Section("Result")
	renderer.Bullet(fmt.Sprintf("updated %s", manifest.FileName))
	_ = ctx
	return nil
}

func runManifestPresetRemove(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	rmFlags := flag.NewFlagSet("manifest preset rm", flag.ContinueOnError)
	var helpFlag bool
	rmFlags.BoolVar(&helpFlag, "help", false, "show help")
	rmFlags.BoolVar(&helpFlag, "h", false, "show help")
	rmFlags.SetOutput(os.Stdout)
	rmFlags.Usage = func() {
		printManifestPresetRmHelp(os.Stdout)
	}
	if err := rmFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printManifestPresetRmHelp(os.Stdout)
		return nil
	}

	file, err := preset.Load(rootDir)
	if err != nil {
		return err
	}

	var names []string
	prompted := false
	if rmFlags.NArg() > 0 {
		names = uniqueStringsPreserve(rmFlags.Args())
		for _, name := range names {
			if err := preset.ValidateName(name); err != nil {
				return err
			}
		}
	} else {
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
		selected, err := ui.PromptMultiSelect("gion manifest preset rm", "preset", choices, theme, useColor)
		if err != nil {
			return err
		}
		prompted = true
		names = uniqueStringsPreserve(selected)
		if len(names) == 0 {
			return nil
		}
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
	if !prompted {
		renderer.Section("Inputs")
		for _, name := range names {
			renderer.Bullet(fmt.Sprintf("preset: %s", name))
		}
		renderer.Blank()
	}
	renderer.Section("Result")
	renderer.Bullet(fmt.Sprintf("updated %s (removed %d presets)", manifest.FileName, len(names)))
	_ = ctx
	return nil
}

func runManifestPresetValidate(ctx context.Context, rootDir string, args []string) error {
	validateFlags := flag.NewFlagSet("manifest preset validate", flag.ContinueOnError)
	validateFlags.SetOutput(os.Stdout)
	var helpFlag bool
	var noPrompt bool
	validateFlags.BoolVar(&helpFlag, "help", false, "show help")
	validateFlags.BoolVar(&helpFlag, "h", false, "show help")
	validateFlags.BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompt (no effect)")
	validateFlags.Usage = func() {
		printManifestPresetValidateHelp(os.Stdout)
	}
	if err := validateFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	_ = noPrompt
	if helpFlag {
		printManifestPresetValidateHelp(os.Stdout)
		return nil
	}
	if validateFlags.NArg() != 0 {
		return fmt.Errorf("usage: gion manifest preset validate [--no-prompt]")
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
		renderer.Bullet(formatManifestPresetValidationIssue(issue))
	}
	_ = ctx
	return fmt.Errorf("manifest preset validation failed")
}

func buildManifestPresetRepoChoices(rootDir string) ([]ui.PromptChoice, error) {
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

func formatManifestPresetValidationIssue(issue preset.ValidationIssue) string {
	kind := strings.TrimSpace(issue.Kind)
	presetName := strings.TrimSpace(issue.Preset)
	repoSpec := strings.TrimSpace(issue.Repo)
	msg := strings.TrimSpace(issue.Message)

	switch kind {
	case manifest.FileName:
		if msg == "" {
			return fmt.Sprintf("%s: missing or unreadable", manifest.FileName)
		}
		return fmt.Sprintf("%s: %s", manifest.FileName, msg)
	case preset.IssueKindInvalidYAML:
		if msg == "" {
			return fmt.Sprintf("%s: invalid yaml", manifest.FileName)
		}
		return fmt.Sprintf("%s: invalid yaml (%s)", manifest.FileName, msg)
	case preset.IssueKindMissingRequired:
		if presetName == "" {
			if msg == "" {
				return "presets: missing required field"
			}
			return fmt.Sprintf("presets: %s", msg)
		}
		if msg == "" || msg == "repos" {
			return fmt.Sprintf("presets.%s.repos: missing or empty", presetName)
		}
		return fmt.Sprintf("presets.%s: %s", presetName, msg)
	case preset.IssueKindDuplicatePreset:
		if presetName == "" {
			return "presets: duplicate preset name"
		}
		return fmt.Sprintf("presets.%s: duplicate preset name", presetName)
	case preset.IssueKindInvalidPresetName:
		if presetName == "" {
			if msg == "" {
				return "presets: invalid preset name"
			}
			return fmt.Sprintf("presets: invalid preset name (%s)", msg)
		}
		if msg == "" {
			return fmt.Sprintf("presets.%s: invalid preset name", presetName)
		}
		return fmt.Sprintf("presets.%s: invalid preset name (%s)", presetName, msg)
	case preset.IssueKindInvalidRepoSpec:
		if presetName == "" {
			if msg == "" {
				return "presets.*.repos: invalid repo spec"
			}
			return fmt.Sprintf("presets.*.repos: invalid repo spec (%s)", msg)
		}
		if repoSpec == "" && msg == "" {
			return fmt.Sprintf("presets.%s.repos: invalid repo spec", presetName)
		}
		if repoSpec == "" {
			return fmt.Sprintf("presets.%s.repos: invalid repo spec (%s)", presetName, msg)
		}
		if msg == "" {
			return fmt.Sprintf("presets.%s.repos: invalid repo spec (%s)", presetName, repoSpec)
		}
		return fmt.Sprintf("presets.%s.repos: invalid repo spec (%s) (%s)", presetName, repoSpec, msg)
	default:
		if msg == "" {
			if presetName == "" {
				return fmt.Sprintf("presets: %s", kind)
			}
			return fmt.Sprintf("presets.%s: %s", presetName, kind)
		}
		if presetName == "" {
			return fmt.Sprintf("presets: %s (%s)", kind, msg)
		}
		return fmt.Sprintf("presets.%s: %s (%s)", presetName, kind, msg)
	}
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
