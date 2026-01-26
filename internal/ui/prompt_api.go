package ui

import (
	"io"
	"strings"

	"github.com/tasuku43/gion/internal/infra/debuglog"
)

func PromptNewWorkspaceInputs(title string, presets []string, presetName string, workspaceID string, theme Theme, useColor bool) (string, string, error) {
	debuglog.SetPrompt("workspace-inputs")
	defer debuglog.ClearPrompt()
	model := newInputsModel(title, presets, presetName, workspaceID, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", "", err
	}
	final := out.(inputsModel)
	if final.err != nil {
		return "", "", final.err
	}
	return strings.TrimSpace(final.preset), strings.TrimSpace(final.workspaceID), nil
}

func PromptWorkspace(title string, workspaces []WorkspaceChoice, theme Theme, useColor bool) (string, error) {
	debuglog.SetPrompt("workspace")
	defer debuglog.ClearPrompt()
	model := newWorkspaceSelectModel(title, workspaces, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", err
	}
	final := out.(workspaceSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.workspaceID), nil
}

func PromptWorkspaceRepoSelect(title string, workspaces []WorkspaceChoice, theme Theme, useColor bool) (string, error) {
	debuglog.SetPrompt("workspace-repo")
	defer debuglog.ClearPrompt()
	model := newWorkspaceRepoSelectModel(title, workspaces, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", err
	}
	final := out.(workspaceRepoSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.selectedPath), nil
}

func PromptWorkspaceRepoSelectWithOutput(title string, workspaces []WorkspaceChoice, theme Theme, useColor bool, out io.Writer) (string, error) {
	debuglog.SetPrompt("workspace-repo")
	defer debuglog.ClearPrompt()
	model := newWorkspaceRepoSelectModel(title, workspaces, theme, useColor)
	finalModel, err := runProgramWithOutput(model, out)
	if err != nil {
		return "", err
	}
	final := finalModel.(workspaceRepoSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.selectedPath), nil
}

func PromptWorkspaceRepoSelectWithIO(title string, workspaces []WorkspaceChoice, theme Theme, useColor bool, in io.Reader, out io.Writer, altScreen bool) (string, error) {
	debuglog.SetPrompt("workspace-repo")
	defer debuglog.ClearPrompt()
	model := newWorkspaceRepoSelectModel(title, workspaces, theme, useColor)
	finalModel, err := runProgramWithIO(model, in, out, altScreen)
	if err != nil {
		return "", err
	}
	final := finalModel.(workspaceRepoSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.selectedPath), nil
}

func PromptWorkspaceWithBlocked(title string, workspaces []WorkspaceChoice, blocked []BlockedChoice, theme Theme, useColor bool) (string, error) {
	debuglog.SetPrompt("workspace")
	defer debuglog.ClearPrompt()
	model := newWorkspaceSelectModelWithBlocked(title, workspaces, blocked, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", err
	}
	final := out.(workspaceSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.workspaceID), nil
}

func PromptWorkspaceMultiSelectWithBlocked(title string, workspaces []WorkspaceChoice, blocked []BlockedChoice, theme Theme, useColor bool) ([]string, error) {
	debuglog.SetPrompt("workspace")
	defer debuglog.ClearPrompt()
	model := newWorkspaceMultiSelectModel(title, workspaces, blocked, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return nil, err
	}
	final := out.(workspaceMultiSelectModel)
	if final.err != nil {
		return nil, final.err
	}
	if final.canceled {
		return nil, nil
	}
	return append([]string(nil), final.selectedIDs...), nil
}

func PromptConfirmInline(label string, theme Theme, useColor bool) (bool, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newConfirmInlineModel(label, theme, useColor, false, nil, nil)
	out, err := runProgram(model)
	if err != nil {
		return false, err
	}
	final := out.(confirmInlineModel)
	if final.err != nil {
		return false, final.err
	}
	return final.value, nil
}

// PromptConfirmInlinePlan renders a single inline confirm prompt line without section headers.
// Intended for embedding inside an existing section (e.g. Plan) after the plan output.
func PromptConfirmInlinePlan(label string, theme Theme, useColor bool) (bool, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newConfirmInlineLineModel(label, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return false, err
	}
	final := out.(confirmInlineLineModel)
	if final.err != nil {
		return false, final.err
	}
	return final.value, nil
}

func PromptConfirmInlineWithRaw(label string, inputsRaw []string, theme Theme, useColor bool) (bool, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newConfirmInlineModelWithRawAfterPrompt(label, theme, useColor, inputsRaw)
	out, err := runProgram(model)
	if err != nil {
		return false, err
	}
	final := out.(confirmInlineModel)
	if final.err != nil {
		return false, final.err
	}
	return final.value, nil
}

func PromptConfirmInlineInfo(label string, theme Theme, useColor bool) (bool, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newConfirmInlineModel(label, theme, useColor, true, nil, nil)
	out, err := runProgram(model)
	if err != nil {
		return false, err
	}
	final := out.(confirmInlineModel)
	if final.err != nil {
		return false, final.err
	}
	return final.value, nil
}

func PromptLabel(label string, theme Theme, useColor bool) string {
	return promptLabel(theme, useColor, label)
}

// PromptInputInline collects a single inline value with an optional default and validation.
// Empty input accepts the default. Validation errors are shown inline and reprompted.
func PromptInputInline(label, defaultValue string, validate func(string) error, theme Theme, useColor bool) (string, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newInputInlineModel(label, defaultValue, validate, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", err
	}
	final := out.(inputInlineModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.value), nil
}

// PromptPresetRepos lets users pick one or more repos from a list with filtering.
// It can also collect a preset name when not provided.
func PromptPresetRepos(title string, presetName string, choices []PromptChoice, theme Theme, useColor bool) (string, []string, error) {
	debuglog.SetPrompt("preset-repos")
	defer debuglog.ClearPrompt()
	model := newPresetRepoSelectModel(title, presetName, choices, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", nil, err
	}
	final := out.(presetRepoSelectModel)
	if final.err != nil {
		return "", nil, final.err
	}
	return strings.TrimSpace(final.presetName), append([]string(nil), final.selected...), nil
}

// PromptPresetName asks for a preset name via text input.
func PromptPresetName(title string, defaultValue string, theme Theme, useColor bool) (string, error) {
	debuglog.SetPrompt("preset-name")
	defer debuglog.ClearPrompt()
	model := newPresetNameModel(title, defaultValue, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", err
	}
	final := out.(presetNameModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.value), nil
}

// PromptChoiceSelect lets users pick a single choice from a list with filtering.
func PromptChoiceSelect(title, label string, choices []PromptChoice, theme Theme, useColor bool) (string, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newChoiceSelectModel(title, label, choices, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", err
	}
	final := out.(choiceSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.value), nil
}

// PromptChoiceSelectWithOutput lets users pick a single choice from a list with filtering.
// It renders the prompt to the provided writer.
func PromptChoiceSelectWithOutput(title, label string, choices []PromptChoice, theme Theme, useColor bool, out io.Writer) (string, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newChoiceSelectModel(title, label, choices, theme, useColor)
	finalModel, err := runProgramWithOutput(model, out)
	if err != nil {
		return "", err
	}
	final := finalModel.(choiceSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.value), nil
}

// PromptMultiSelect lets users pick one or more choices from a list with filtering.
func PromptMultiSelect(title, label string, choices []PromptChoice, theme Theme, useColor bool) ([]string, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newMultiSelectModel(title, label, choices, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return nil, err
	}
	final := out.(multiSelectModel)
	if final.err != nil {
		return nil, final.err
	}
	return append([]string(nil), final.selectedValues...), nil
}

func PromptIssueSelectWithBranches(title, label string, choices []PromptChoice, validateBranch func(string) error, theme Theme, useColor bool) ([]IssueSelection, error) {
	debuglog.SetPrompt(label)
	defer debuglog.ClearPrompt()
	model := newIssueBranchSelectModel(title, label, choices, validateBranch, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return nil, err
	}
	final := out.(issueBranchSelectModel)
	if final.err != nil {
		return nil, final.err
	}
	return append([]IssueSelection(nil), final.selectedIssues...), nil
}

func PromptCreateFlow(title string, startMode string, defaultWorkspaceID string, presetName string, presets []string, presetErr error, repoChoices []PromptChoice, repoErr error, reviewRepos []PromptChoice, issueRepos []PromptChoice, loadReview func(string) ([]PromptChoice, error), loadIssue func(string) ([]PromptChoice, error), loadPresetRepos func(string) ([]string, error), onReposResolved func([]string), validateBranch func(string) error, validateWorkspaceID func(string) error, theme Theme, useColor bool, selectedRepo string) (string, string, string, string, []string, string, []string, string, []IssueSelection, string, error) {
	debuglog.SetPrompt("create-flow")
	defer debuglog.ClearPrompt()
	model := newCreateFlowModel(title, presets, presetErr, repoChoices, repoErr, defaultWorkspaceID, presetName, reviewRepos, issueRepos, loadReview, loadIssue, loadPresetRepos, onReposResolved, validateBranch, validateWorkspaceID, theme, useColor, startMode, selectedRepo)
	if model.err != nil {
		return "", "", "", "", nil, "", nil, "", nil, "", model.err
	}
	out, err := runProgram(model)
	if err != nil {
		return "", "", "", "", nil, "", nil, "", nil, "", err
	}
	final := out.(createFlowModel)
	if final.err != nil {
		return "", "", "", "", nil, "", nil, "", nil, "", final.err
	}
	return final.mode, final.presetName(), final.workspaceID(), final.description, append([]string(nil), final.branches...), final.reviewRepo, append([]string(nil), final.reviewPRs...), final.issueRepo, append([]IssueSelection(nil), final.issueIssues...), final.repoSelected, nil
}

func WorkspaceChoiceLines(items []WorkspaceChoice, cursor int, useColor bool, theme Theme) []string {
	var lines []string
	builder := &strings.Builder{}
	renderWorkspaceChoiceList(builder, items, cursor, listMaxLines(0, len(items), 0), useColor, theme)
	lines = append(lines, splitLines(builder.String())...)
	return lines
}

func WorkspaceChoiceConfirmLines(items []WorkspaceChoice, useColor bool, theme Theme) []string {
	var lines []string
	builder := &strings.Builder{}
	renderWorkspaceChoiceConfirmList(builder, items, useColor, theme)
	lines = append(lines, splitLines(builder.String())...)
	return lines
}
