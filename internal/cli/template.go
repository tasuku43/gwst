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
	"github.com/tasuku43/gwst/internal/core/output"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/template"
	"github.com/tasuku43/gwst/internal/ui"
)

func runTemplate(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printTemplateHelp(os.Stdout)
		return nil
	}
	switch args[0] {
	case "ls":
		return runTemplateList(ctx, rootDir, args[1:])
	case "add":
		return runTemplateAdd(ctx, rootDir, args[1:], noPrompt)
	case "rm":
		return runTemplateRemove(ctx, rootDir, args[1:], noPrompt)
	case "validate":
		return runTemplateValidate(ctx, rootDir, args[1:])
	default:
		return fmt.Errorf("unknown template subcommand: %s", args[0])
	}
}

func runTemplateList(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printTemplateLsHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst template ls")
	}
	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}
	names := template.Names(file)
	writeTemplateListText(file, names)
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

func runTemplateAdd(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	addFlags := flag.NewFlagSet("template add", flag.ContinueOnError)
	var helpFlag bool
	var repos stringSliceFlag
	addFlags.Var(&repos, "repo", "repo spec (repeatable)")
	addFlags.BoolVar(&helpFlag, "help", false, "show help")
	addFlags.BoolVar(&helpFlag, "h", false, "show help")
	addFlags.SetOutput(os.Stdout)
	addFlags.Usage = func() {
		printTemplateAddHelp(os.Stdout)
	}
	if err := addFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printTemplateAddHelp(os.Stdout)
		return nil
	}
	if addFlags.NArg() > 1 {
		return fmt.Errorf("usage: gwst template add [<name>] [--repo <repo> ...]")
	}

	name := ""
	if addFlags.NArg() == 1 {
		name = addFlags.Arg(0)
	}

	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}

	repoSpecs := template.NormalizeRepos(repos)

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())

	if strings.TrimSpace(name) == "" && len(repoSpecs) == 0 {
		if noPrompt {
			return fmt.Errorf("template name and repos are required with --no-prompt")
		}
		choices, err := buildTemplateRepoChoices(rootDir)
		if err != nil {
			return err
		}
		if len(choices) == 0 {
			return fmt.Errorf("no repos found; run gwst repo get first")
		}
		name, repoSpecs, err = ui.PromptTemplateRepos("gwst template add", name, choices, theme, useColor)
		if err != nil {
			return err
		}
		repoSpecs = template.NormalizeRepos(repoSpecs)
	} else {
		if strings.TrimSpace(name) == "" {
			if noPrompt {
				return fmt.Errorf("template name is required with --no-prompt")
			}
			name, err = ui.PromptTemplateName("gwst template add", "", theme, useColor)
			if err != nil {
				return err
			}
		}
		if len(repoSpecs) == 0 {
			if noPrompt {
				return fmt.Errorf("repos are required with --no-prompt")
			}
			choices, err := buildTemplateRepoChoices(rootDir)
			if err != nil {
				return err
			}
			if len(choices) == 0 {
				return fmt.Errorf("no repos found; run gwst repo get first")
			}
			var selected []string
			name, selected, err = ui.PromptTemplateRepos("gwst template add", name, choices, theme, useColor)
			if err != nil {
				return err
			}
			repoSpecs = template.NormalizeRepos(selected)
		}
	}

	if err := template.ValidateName(name); err != nil {
		return err
	}
	if _, exists := file.Templates[name]; exists {
		return fmt.Errorf("template already exists: %s", name)
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

	if file.Templates == nil {
		file.Templates = map[string]template.Template{}
	}
	file.Templates[name] = template.Template{Repos: repoSpecs}

	if err := template.Save(rootDir, file); err != nil {
		return err
	}

	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	renderer.Section("Result")
	renderer.Bullet(name)
	var reposDisplay []string
	for _, repoSpec := range repoSpecs {
		reposDisplay = append(reposDisplay, displayTemplateRepo(repoSpec))
	}
	renderTreeLines(renderer, reposDisplay, treeLineNormal)
	renderSuggestions(renderer, useColor, []string{
		"gwst create --template",
		"gwst create --template <name>",
	})
	return nil
}

func runTemplateRemove(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	rmFlags := flag.NewFlagSet("template rm", flag.ContinueOnError)
	var helpFlag bool
	rmFlags.BoolVar(&helpFlag, "help", false, "show help")
	rmFlags.BoolVar(&helpFlag, "h", false, "show help")
	rmFlags.SetOutput(os.Stdout)
	rmFlags.Usage = func() {
		printTemplateRmHelp(os.Stdout)
	}
	if err := rmFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printTemplateRmHelp(os.Stdout)
		return nil
	}

	file, err := template.Load(rootDir)
	if err != nil {
		return err
	}

	var names []string
	showInputs := true
	if rmFlags.NArg() > 0 {
		names = uniqueStringsPreserve(rmFlags.Args())
		for _, name := range names {
			if err := template.ValidateName(name); err != nil {
				return err
			}
		}
	} else {
		showInputs = false
		if noPrompt {
			return fmt.Errorf("template name is required with --no-prompt")
		}
		templates := template.Names(file)
		if len(templates) == 0 {
			return fmt.Errorf("no templates found in %s", filepath.Join(rootDir, template.FileName))
		}
		var choices []ui.PromptChoice
		for _, name := range templates {
			choices = append(choices, ui.PromptChoice{Label: name, Value: name})
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		selected, err := ui.PromptMultiSelect("gwst template rm", "template", choices, theme, useColor)
		if err != nil {
			return err
		}
		names = uniqueStringsPreserve(selected)
	}

	for _, name := range names {
		if _, exists := file.Templates[name]; !exists {
			return fmt.Errorf("template not found: %s", name)
		}
	}

	for _, name := range names {
		delete(file.Templates, name)
	}

	if err := template.Save(rootDir, file); err != nil {
		return err
	}

	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)
	if showInputs {
		renderer.Section("Inputs")
		renderer.Bullet("templates")
		renderTreeLines(renderer, names, treeLineNormal)
		renderer.Blank()
	}
	renderer.Section("Steps")
	for i, name := range names {
		output.Step(formatStepWithIndex("remove template", name, relPath(rootDir, filepath.Join(rootDir, template.FileName)), i+1, len(names)))
	}
	renderer.Blank()
	renderer.Section("Result")
	for _, name := range names {
		renderer.Bullet(fmt.Sprintf("%s removed", name))
	}
	return nil
}

func runTemplateValidate(ctx context.Context, rootDir string, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printTemplateValidateHelp(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: gwst template validate")
	}
	result, err := template.Validate(rootDir)
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
		details := templateIssueDetails(issue, result.Path)
		if len(details) > 0 {
			renderTreeLines(renderer, details, treeLineError)
		}
	}
	return fmt.Errorf("template validation failed")
}

func buildTemplateRepoChoices(rootDir string) ([]ui.PromptChoice, error) {
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
