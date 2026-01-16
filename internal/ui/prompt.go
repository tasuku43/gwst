package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tasuku43/gws/internal/core/output"
)

var ErrPromptCanceled = errors.New("prompt canceled")

type PromptChoice struct {
	Label string
	Value string
}

type IssueSelection struct {
	Value  string
	Branch string
}

type createFlowStage int

const (
	createStageMode createFlowStage = iota
	createStageTemplate
	createStageTemplateDesc
	createStageTemplateBranch
	createStageTemplateBranchConfirm
	createStageReviewRepo
	createStageReviewPRs
	createStageIssueRepo
	createStageIssueIssues
	createStageRepoSelect
	createStageRepoWorkspace
)

type WorkspaceChoice struct {
	ID            string
	Description   string
	Repos         []PromptChoice
	Warning       string
	WarningStrong bool
}

type BlockedChoice struct {
	Label string
}

func PromptNewWorkspaceInputs(title string, templates []string, templateName string, workspaceID string, theme Theme, useColor bool) (string, string, error) {
	model := newInputsModel(title, templates, templateName, workspaceID, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", "", err
	}
	final := out.(inputsModel)
	if final.err != nil {
		return "", "", final.err
	}
	return strings.TrimSpace(final.template), strings.TrimSpace(final.workspaceID), nil
}

func PromptWorkspaceAndRepo(title string, workspaces []WorkspaceChoice, repos []PromptChoice, workspaceID, repoSpec string, theme Theme, useColor bool) (string, string, error) {
	model := newAddInputsModel(title, workspaces, repos, workspaceID, repoSpec, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", "", err
	}
	final := out.(addInputsModel)
	if final.err != nil {
		return "", "", final.err
	}
	return strings.TrimSpace(final.workspaceID), strings.TrimSpace(final.repoSpec), nil
}

func PromptWorkspace(title string, workspaces []WorkspaceChoice, theme Theme, useColor bool) (string, error) {
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

func PromptWorkspaceWithBlocked(title string, workspaces []WorkspaceChoice, blocked []BlockedChoice, theme Theme, useColor bool) (string, error) {
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

func PromptConfirmInlineInfo(label string, theme Theme, useColor bool) (bool, error) {
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

type inputsStage int

const (
	stageTemplate inputsStage = iota
	stageWorkspace
)

type inputsModel struct {
	title       string
	templates   []string
	template    string
	workspaceID string
	label       string

	stage     inputsStage
	theme     Theme
	useColor  bool
	search    textinput.Model
	idInput   textinput.Model
	filtered  []string
	cursor    int
	err       error
	errorLine string
	done      bool
	height    int
}

type createFlowModel struct {
	stage createFlowStage
	mode  string

	templates []string
	tmplErr   error

	modeInput textinput.Model
	filtered  []PromptChoice
	cursor    int
	err       error

	templateModel      inputsModel
	templateRepos      []string
	description        string
	descInput          textinput.Model
	branchInput        textinput.Model
	branchIndex        int
	branches           []string
	usedBranches       map[string]int
	pendingBranch      string
	confirmModel       confirmInlineModel
	errorLine          string
	reviewRepos        []PromptChoice
	issueRepos         []PromptChoice
	repoChoices        []PromptChoice
	repoErr            error
	defaultWorkspaceID string

	height       int
	reviewRepo   string
	issueRepo    string
	reviewPRs    []string
	issueIssues  []IssueSelection
	repoSelected string

	reviewRepoModel   choiceSelectModel
	reviewPRModel     multiSelectModel
	issueRepoModel    choiceSelectModel
	issueIssueModel   issueBranchSelectModel
	repoSelectModel   choiceSelectModel
	loadReviewPRs     func(string) ([]PromptChoice, error)
	loadIssueChoices  func(string) ([]PromptChoice, error)
	loadTemplateRepos func(string) ([]string, error)
	validateBranch    func(string) error

	theme    Theme
	useColor bool
}

func newCreateFlowModel(title string, templates []string, tmplErr error, repoChoices []PromptChoice, repoErr error, defaultWorkspaceID string, templateName string, reviewRepos []PromptChoice, issueRepos []PromptChoice, loadReview func(string) ([]PromptChoice, error), loadIssue func(string) ([]PromptChoice, error), loadTemplateRepos func(string) ([]string, error), validateBranch func(string) error, theme Theme, useColor bool, startMode string, selectedRepo string) createFlowModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := createFlowModel{
		stage:              createStageMode,
		templates:          templates,
		tmplErr:            tmplErr,
		repoChoices:        repoChoices,
		repoErr:            repoErr,
		defaultWorkspaceID: defaultWorkspaceID,
		modeInput:          input,
		reviewRepos:        reviewRepos,
		issueRepos:         issueRepos,
		loadReviewPRs:      loadReview,
		loadIssueChoices:   loadIssue,
		loadTemplateRepos:  loadTemplateRepos,
		validateBranch:     validateBranch,
		theme:              theme,
		useColor:           useColor,
	}
	m.repoSelected = strings.TrimSpace(selectedRepo)
	templateName = strings.TrimSpace(templateName)
	if strings.TrimSpace(startMode) != "" {
		if startMode == "repo" && m.repoSelected != "" {
			m.mode = "repo"
			m.templateRepos = []string{m.repoSelected}
			m.templateModel = newInputsModelWithLabel("gws create", nil, m.repoSelected, m.defaultWorkspaceID, "repo", m.theme, m.useColor)
			m.stage = createStageRepoWorkspace
		} else {
			m.startMode(startMode, templateName)
		}
	}
	if m.stage == createStageMode {
		m.filtered = m.filterModes()
	}
	return m
}

func (m createFlowModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *createFlowModel) startMode(mode, templateName string) {
	switch mode {
	case "template":
		if m.tmplErr != nil {
			m.err = m.tmplErr
			return
		}
		if len(m.templates) == 0 {
			m.err = fmt.Errorf("no templates found")
			return
		}
		m.mode = mode
		m.stage = createStageTemplate
		m.templateModel = newInputsModel("gws create", m.templates, templateName, m.defaultWorkspaceID, m.theme, m.useColor)
	case "review":
		if len(m.reviewRepos) == 0 {
			m.err = fmt.Errorf("no GitHub repos found")
			return
		}
		m.mode = mode
		m.stage = createStageReviewRepo
		m.reviewRepoModel = newChoiceSelectModel("gws create", "repo", m.reviewRepos, m.theme, m.useColor)
	case "issue":
		if len(m.issueRepos) == 0 {
			m.err = fmt.Errorf("no repos with supported hosts found")
			return
		}
		m.mode = mode
		m.stage = createStageIssueRepo
		m.issueRepoModel = newChoiceSelectModel("gws create", "repo", m.issueRepos, m.theme, m.useColor)
	case "repo":
		if m.repoErr != nil {
			m.err = m.repoErr
			return
		}
		if len(m.repoChoices) == 0 {
			m.err = fmt.Errorf("no repos found")
			return
		}
		m.mode = mode
		m.stage = createStageRepoSelect
		m.repoSelectModel = newChoiceSelectModel("gws create", "repo", m.repoChoices, m.theme, m.useColor)
	default:
		m.err = fmt.Errorf("unknown mode: %s", mode)
	}
}

func (m *createFlowModel) beginDescriptionStage() {
	m.descInput = textinput.New()
	m.descInput.Prompt = ""
	m.descInput.Placeholder = "description"
	m.descInput.Focus()
	if m.useColor {
		m.descInput.PlaceholderStyle = m.theme.Muted
	}
	m.stage = createStageTemplateDesc
}

func (m createFlowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = size.Height
		switch m.stage {
		case createStageTemplate:
			model, _ := m.templateModel.Update(msg)
			m.templateModel = model.(inputsModel)
		case createStageRepoWorkspace:
			model, _ := m.templateModel.Update(msg)
			m.templateModel = model.(inputsModel)
		case createStageReviewRepo:
			model, _ := m.reviewRepoModel.Update(msg)
			m.reviewRepoModel = model.(choiceSelectModel)
		case createStageReviewPRs:
			model, _ := m.reviewPRModel.Update(msg)
			m.reviewPRModel = model.(multiSelectModel)
		case createStageIssueRepo:
			model, _ := m.issueRepoModel.Update(msg)
			m.issueRepoModel = model.(choiceSelectModel)
		case createStageIssueIssues:
			model, _ := m.issueIssueModel.Update(msg)
			m.issueIssueModel = model.(issueBranchSelectModel)
		case createStageRepoSelect:
			model, _ := m.repoSelectModel.Update(msg)
			m.repoSelectModel = model.(choiceSelectModel)
		}
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyUp:
			if m.stage == createStageMode && m.cursor > 0 {
				m.cursor--
				return m, nil
			}
		case tea.KeyDown:
			if m.stage == createStageMode && m.cursor < len(m.filtered)-1 {
				m.cursor++
				return m, nil
			}
		case tea.KeyEnter:
			if m.stage == createStageMode {
				if len(m.filtered) == 0 {
					return m, nil
				}
				m.mode = m.filtered[m.cursor].Value
				switch m.mode {
				case "template":
					if m.tmplErr != nil {
						m.err = m.tmplErr
						return m, tea.Quit
					}
					m.stage = createStageTemplate
					m.templateModel = newInputsModel("gws create", m.templates, "", "", m.theme, m.useColor)
				case "review":
					if len(m.reviewRepos) == 0 {
						m.err = fmt.Errorf("no GitHub repos found")
						return m, tea.Quit
					}
					m.stage = createStageReviewRepo
					m.reviewRepoModel = newChoiceSelectModel("gws create", "repo", m.reviewRepos, m.theme, m.useColor)
				case "issue":
					if len(m.issueRepos) == 0 {
						m.err = fmt.Errorf("no repos with supported hosts found")
						return m, tea.Quit
					}
					m.stage = createStageIssueRepo
					m.issueRepoModel = newChoiceSelectModel("gws create", "repo", m.issueRepos, m.theme, m.useColor)
				case "repo":
					if m.repoErr != nil {
						m.err = m.repoErr
						return m, tea.Quit
					}
					if len(m.repoChoices) == 0 {
						m.err = fmt.Errorf("no repos found")
						return m, tea.Quit
					}
					m.stage = createStageRepoSelect
					m.repoSelectModel = newChoiceSelectModel("gws create", "repo", m.repoChoices, m.theme, m.useColor)
				default:
					m.err = fmt.Errorf("unknown mode: %s", m.mode)
					return m, tea.Quit
				}
				return m, nil
			}
		}
	}

	if m.stage == createStageTemplate {
		model, _ := m.templateModel.Update(msg)
		m.templateModel = model.(inputsModel)
		if m.templateModel.done {
			repos, err := m.loadTemplateRepos(m.templateName())
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			m.templateRepos = repos
			m.beginDescriptionStage()
			return m, nil
		}
		return m, nil
	}

	if m.stage == createStageTemplateDesc {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				m.err = ErrPromptCanceled
				return m, tea.Quit
			case tea.KeyEnter:
				m.description = strings.TrimSpace(m.descInput.Value())
				if len(m.templateRepos) == 0 {
					return m, tea.Quit
				}
				m.branches = make([]string, len(m.templateRepos))
				m.usedBranches = map[string]int{}
				m.branchIndex = 0
				m.branchInput = textinput.New()
				m.branchInput.Prompt = ""
				m.branchInput.Placeholder = m.workspaceID()
				m.branchInput.SetValue(m.workspaceID())
				m.branchInput.Focus()
				if m.useColor {
					m.branchInput.PlaceholderStyle = m.theme.Muted
				}
				m.stage = createStageTemplateBranch
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.descInput, cmd = m.descInput.Update(msg)
		return m, cmd
	}

	if m.stage == createStageTemplateBranch {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				m.err = ErrPromptCanceled
				return m, tea.Quit
			case tea.KeyEnter:
				value := strings.TrimSpace(m.branchInput.Value())
				if value == "" {
					value = m.workspaceID()
				}
				if m.validateBranch != nil {
					if err := m.validateBranch(value); err != nil {
						m.errorLine = err.Error()
						return m, nil
					}
				}
				if prevIndex, exists := m.usedBranches[value]; exists {
					label := fmt.Sprintf("branch %q already used for repo #%d; use again?", value, prevIndex+1)
					m.pendingBranch = value
					m.confirmModel = newConfirmInlineModel(label, m.theme, m.useColor, false, nil, nil)
					m.stage = createStageTemplateBranchConfirm
					return m, nil
				}
				m.branches[m.branchIndex] = value
				m.usedBranches[value] = m.branchIndex
				m.branchIndex++
				m.errorLine = ""
				if m.branchIndex >= len(m.templateRepos) {
					return m, tea.Quit
				}
				m.branchInput.SetValue(m.workspaceID())
				m.branchInput.CursorEnd()
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.branchInput, cmd = m.branchInput.Update(msg)
		if strings.TrimSpace(m.branchInput.Value()) != "" {
			m.errorLine = ""
		}
		return m, cmd
	}

	if m.stage == createStageTemplateBranchConfirm {
		model, _ := m.confirmModel.Update(msg)
		m.confirmModel = model.(confirmInlineModel)
		if m.confirmModel.done {
			if m.confirmModel.value {
				m.branches[m.branchIndex] = m.pendingBranch
				m.usedBranches[m.pendingBranch] = m.branchIndex
				m.branchIndex++
				if m.branchIndex >= len(m.templateRepos) {
					return m, tea.Quit
				}
				m.branchInput.SetValue(m.workspaceID())
				m.branchInput.CursorEnd()
			}
			m.pendingBranch = ""
			m.confirmModel = confirmInlineModel{}
			m.stage = createStageTemplateBranch
		}
		return m, nil
	}

	if m.stage == createStageReviewRepo {
		model, _ := m.reviewRepoModel.Update(msg)
		m.reviewRepoModel = model.(choiceSelectModel)
		if m.reviewRepoModel.done {
			m.reviewRepo = m.reviewRepoModel.value
			choices, err := m.loadReviewPRs(m.reviewRepo)
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			if len(choices) == 0 {
				m.err = fmt.Errorf("no pull requests found")
				return m, tea.Quit
			}
			m.reviewPRModel = newMultiSelectModel("gws create", "pull request", choices, m.theme, m.useColor)
			m.stage = createStageReviewPRs
		}
		return m, nil
	}

	if m.stage == createStageReviewPRs {
		model, _ := m.reviewPRModel.Update(msg)
		m.reviewPRModel = model.(multiSelectModel)
		if m.reviewPRModel.done {
			m.reviewPRs = append([]string(nil), m.reviewPRModel.selectedValues...)
			return m, tea.Quit
		}
		return m, nil
	}

	if m.stage == createStageIssueRepo {
		model, _ := m.issueRepoModel.Update(msg)
		m.issueRepoModel = model.(choiceSelectModel)
		if m.issueRepoModel.done {
			m.issueRepo = m.issueRepoModel.value
			choices, err := m.loadIssueChoices(m.issueRepo)
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			if len(choices) == 0 {
				m.err = fmt.Errorf("no issues found")
				return m, tea.Quit
			}
			m.issueIssueModel = newIssueBranchSelectModel("gws create", "issue", choices, m.validateBranch, m.theme, m.useColor)
			m.stage = createStageIssueIssues
		}
		return m, nil
	}

	if m.stage == createStageIssueIssues {
		model, _ := m.issueIssueModel.Update(msg)
		m.issueIssueModel = model.(issueBranchSelectModel)
		if m.issueIssueModel.done {
			m.issueIssues = append([]IssueSelection(nil), m.issueIssueModel.selectedIssues...)
			return m, tea.Quit
		}
		return m, nil
	}

	if m.stage == createStageRepoSelect {
		model, _ := m.repoSelectModel.Update(msg)
		m.repoSelectModel = model.(choiceSelectModel)
		if m.repoSelectModel.done {
			m.repoSelected = m.repoSelectModel.value
			m.templateRepos = []string{m.repoSelected}
			m.templateModel = newInputsModelWithLabel("gws create", nil, m.repoSelected, m.defaultWorkspaceID, "repo", m.theme, m.useColor)
			m.stage = createStageRepoWorkspace
		}
		return m, nil
	}

	if m.stage == createStageRepoWorkspace {
		model, _ := m.templateModel.Update(msg)
		m.templateModel = model.(inputsModel)
		if m.templateModel.done {
			m.beginDescriptionStage()
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.modeInput, cmd = m.modeInput.Update(msg)
	m.filtered = m.filterModes()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m createFlowModel) View() string {
	if m.stage == createStageTemplate {
		return m.templateModel.View()
	}
	if m.stage == createStageTemplateDesc {
		frame := NewFrame(m.theme, m.useColor)
		labelSelection := promptLabel(m.theme, m.useColor, m.selectionLabel())
		labelWorkspace := promptLabel(m.theme, m.useColor, "workspace id")
		labelDesc := promptLabel(m.theme, m.useColor, "description")
		frame.SetInputsPrompt(
			fmt.Sprintf("%s: %s", labelSelection, m.selectionValue()),
			fmt.Sprintf("%s: %s", labelWorkspace, m.workspaceID()),
			fmt.Sprintf("%s: %s", labelDesc, m.descInput.View()),
		)
		return frame.Render()
	}
	if m.stage == createStageTemplateBranch {
		frame := NewFrame(m.theme, m.useColor)
		labelSelection := promptLabel(m.theme, m.useColor, m.selectionLabel())
		labelWorkspace := promptLabel(m.theme, m.useColor, "workspace id")
		labelDesc := promptLabel(m.theme, m.useColor, "description")
		repoLabel := fmt.Sprintf("repo #%d", m.branchIndex+1)
		if m.branchIndex < len(m.templateRepos) {
			repoLabel = fmt.Sprintf("repo #%d (%s)", m.branchIndex+1, m.templateRepos[m.branchIndex])
		}
		labelBranch := promptLabel(m.theme, m.useColor, fmt.Sprintf("branch for %s", repoLabel))
		lines := []string{
			fmt.Sprintf("%s: %s", labelSelection, m.selectionValue()),
			fmt.Sprintf("%s: %s", labelWorkspace, m.workspaceID()),
		}
		if m.description != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", labelDesc, m.description))
		}
		lines = append(lines, fmt.Sprintf("%s: %s", labelBranch, m.branchInput.View()))
		frame.SetInputsPrompt(lines...)
		if m.errorLine != "" {
			frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), m.errorLine))
		}
		return frame.Render()
	}
	if m.stage == createStageTemplateBranchConfirm {
		return m.confirmModel.View()
	}
	if m.stage == createStageReviewRepo {
		return m.reviewRepoModel.View()
	}
	if m.stage == createStageReviewPRs {
		labelRepo := promptLabel(m.theme, m.useColor, "repo")
		return renderMultiSelectFrame(m.reviewPRModel, m.reviewPRModel.height, fmt.Sprintf("%s: %s", labelRepo, m.reviewRepo))
	}
	if m.stage == createStageIssueRepo {
		return m.issueRepoModel.View()
	}
	if m.stage == createStageIssueIssues {
		labelRepo := promptLabel(m.theme, m.useColor, "repo")
		if m.issueIssueModel.stage == issueBranchStageEdit {
			return renderIssueBranchEditFrame(m.issueIssueModel, fmt.Sprintf("%s: %s", labelRepo, m.issueRepo))
		}
		return renderMultiSelectFrame(multiSelectModel{
			title:     m.issueIssueModel.title,
			label:     m.issueIssueModel.label,
			choices:   m.issueIssueModel.choices,
			filtered:  m.issueIssueModel.filtered,
			selected:  m.issueIssueModel.selected,
			cursor:    m.issueIssueModel.cursor,
			errorLine: m.issueIssueModel.errorLine,
			theme:     m.issueIssueModel.theme,
			useColor:  m.issueIssueModel.useColor,
			input:     m.issueIssueModel.input,
			height:    m.issueIssueModel.height,
		}, m.issueIssueModel.height, fmt.Sprintf("%s: %s", labelRepo, m.issueRepo))
	}
	if m.stage == createStageRepoSelect {
		return m.repoSelectModel.View()
	}
	if m.stage == createStageRepoWorkspace {
		return m.templateModel.View()
	}

	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "mode")
	frame.SetInputsPrompt(fmt.Sprintf("%s: %s", label, m.modeInput.View()))
	maxLines := listMaxLines(m.height, 1, 0)
	rawLines := collectLines(func(b *strings.Builder) {
		renderRepoChoiceList(b, m.filtered, m.cursor, maxLines, m.useColor, m.theme)
	})
	frame.AppendInputsRaw(rawLines...)
	return frame.Render()
}

func (m createFlowModel) filterModes() []PromptChoice {
	q := strings.ToLower(strings.TrimSpace(m.modeInput.Value()))
	choices := []PromptChoice{
		{Label: "template", Value: "template"},
		{Label: "repo", Value: "repo"},
		{Label: "review", Value: "review"},
		{Label: "issue", Value: "issue"},
	}
	if q == "" {
		return choices
	}
	var out []PromptChoice
	for _, item := range choices {
		if strings.Contains(strings.ToLower(item.Label), q) || strings.Contains(strings.ToLower(item.Value), q) {
			out = append(out, item)
		}
	}
	return out
}

func (m createFlowModel) templateName() string {
	return strings.TrimSpace(m.templateModel.template)
}

func (m createFlowModel) workspaceID() string {
	return m.templateModel.currentWorkspaceID()
}

func (m createFlowModel) selectionLabel() string {
	if m.mode == "repo" {
		return "repo"
	}
	return "template"
}

func (m createFlowModel) selectionValue() string {
	if m.mode == "repo" {
		return m.repoSelected
	}
	return m.templateName()
}

func newInputsModel(title string, templates []string, templateName string, workspaceID string, theme Theme, useColor bool) inputsModel {
	return newInputsModelWithLabel(title, templates, templateName, workspaceID, "template", theme, useColor)
}

func newInputsModelWithLabel(title string, templates []string, templateName string, workspaceID string, label string, theme Theme, useColor bool) inputsModel {
	search := textinput.New()
	search.Prompt = ""
	search.Placeholder = "search"
	search.Focus()
	if useColor {
		search.PlaceholderStyle = theme.Muted
	}

	idInput := textinput.New()
	idInput.Prompt = ""
	idInput.Placeholder = "type here"
	if useColor {
		idInput.PlaceholderStyle = theme.Muted
	}

	stage := stageTemplate
	if strings.TrimSpace(templateName) != "" {
		stage = stageWorkspace
		idInput.Focus()
	} else {
		search.Focus()
	}
	if workspaceID != "" {
		idInput.SetValue(workspaceID)
	}

	m := inputsModel{
		title:       title,
		templates:   templates,
		template:    templateName,
		workspaceID: workspaceID,
		label:       label,
		stage:       stage,
		theme:       theme,
		useColor:    useColor,
		search:      search,
		idInput:     idInput,
	}
	m.filtered = m.filterItems()
	return m
}

func (m inputsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyEnter:
			if m.stage == stageTemplate {
				if len(m.filtered) == 0 {
					return m, nil
				}
				m.template = m.filtered[m.cursor]
				if strings.TrimSpace(m.workspaceID) != "" {
					m.done = true
					return m, tea.Quit
				}
				m.stage = stageWorkspace
				m.idInput.Focus()
				return m, nil
			}
			if m.stage == stageWorkspace {
				value := strings.TrimSpace(m.idInput.Value())
				if value == "" {
					m.errorLine = "required"
					return m, nil
				}
				m.workspaceID = value
				m.done = true
				return m, tea.Quit
			}
		case tea.KeyUp:
			if m.stage == stageTemplate && m.cursor > 0 {
				m.cursor--
				return m, nil
			}
		case tea.KeyDown:
			if m.stage == stageTemplate && m.cursor < len(m.filtered)-1 {
				m.cursor++
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	if m.stage == stageTemplate {
		m.search, cmd = m.search.Update(msg)
		m.filtered = m.filterItems()
		if m.cursor >= len(m.filtered) {
			m.cursor = max(0, len(m.filtered)-1)
		}
	} else {
		m.idInput, cmd = m.idInput.Update(msg)
		if strings.TrimSpace(m.idInput.Value()) != "" {
			m.errorLine = ""
		}
	}
	return m, cmd
}

func (m inputsModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := strings.TrimSpace(m.label)
	if label == "" {
		label = "template"
	}
	labelTemplate := promptLabel(m.theme, m.useColor, label)
	var promptLines []string
	if m.stage == stageTemplate {
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", labelTemplate, m.search.View()))
	} else {
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", labelTemplate, m.template))
	}

	if m.stage == stageWorkspace {
		label := promptLabel(m.theme, m.useColor, "workspace id")
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", label, m.idInput.View()))
	} else if strings.TrimSpace(m.workspaceID) != "" {
		label := promptLabel(m.theme, m.useColor, "workspace id")
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", label, m.workspaceID))
	}
	frame.SetInputsPrompt(promptLines...)

	if m.stage == stageTemplate {
		maxLines := listMaxLines(m.height, len(promptLines), 0)
		rawLines := collectLines(func(b *strings.Builder) {
			renderChoiceList(b, m.filtered, m.cursor, maxLines, m.useColor, m.theme)
		})
		frame.AppendInputsRaw(rawLines...)
	}

	if m.stage == stageWorkspace && m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		frame.AppendInputsRaw(fmt.Sprintf("%s%s %s", output.Indent+output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}

	return frame.Render()
}

func (m inputsModel) currentWorkspaceID() string {
	if m.stage == stageWorkspace {
		if value := strings.TrimSpace(m.idInput.Value()); value != "" {
			return value
		}
	}
	return strings.TrimSpace(m.workspaceID)
}

func formatInputsHeader(title, templateName, workspaceID string) string {
	var parts []string
	if strings.TrimSpace(templateName) != "" {
		parts = append(parts, fmt.Sprintf("template: %s", templateName))
	}
	if strings.TrimSpace(workspaceID) != "" {
		parts = append(parts, fmt.Sprintf("workspace id: %s", workspaceID))
	}
	if len(parts) == 0 {
		return title
	}
	return fmt.Sprintf("%s (%s)", title, strings.Join(parts, ", "))
}

func (m inputsModel) filterItems() []string {
	q := strings.ToLower(strings.TrimSpace(m.search.Value()))
	if q == "" {
		return append([]string(nil), m.templates...)
	}
	var out []string
	for _, item := range m.templates {
		if strings.Contains(strings.ToLower(item), q) {
			out = append(out, item)
		}
	}
	return out
}

type confirmInlineModel struct {
	label        string
	theme        Theme
	useColor     bool
	useInfo      bool
	inputsPrompt []string
	inputsRaw    []string
	input        textinput.Model
	value        bool
	err          error
	done         bool
}

func newConfirmInlineModel(label string, theme Theme, useColor bool, useInfo bool, inputsPrompt []string, inputsRaw []string) confirmInlineModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "y/n"
	ti.Focus()
	if useColor {
		ti.PlaceholderStyle = theme.Muted
	}
	return confirmInlineModel{
		label:        label,
		theme:        theme,
		useColor:     useColor,
		useInfo:      useInfo,
		inputsPrompt: append([]string(nil), inputsPrompt...),
		inputsRaw:    append([]string(nil), inputsRaw...),
		input:        ti,
	}
}

func (m confirmInlineModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m confirmInlineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyEnter:
			value := strings.ToLower(strings.TrimSpace(m.input.Value()))
			switch value {
			case "y", "yes":
				m.value = true
				m.done = true
				return m, tea.Quit
			case "n", "no":
				m.value = false
				m.done = true
				return m, tea.Quit
			default:
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m confirmInlineModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, m.label)
	line := fmt.Sprintf("%s (y/n): %s", label, m.input.View())
	if m.useInfo {
		frame.SetInfoPrompt(line)
	} else {
		if len(m.inputsPrompt) > 0 {
			frame.SetInputsPrompt(m.inputsPrompt...)
		}
		if len(m.inputsRaw) > 0 {
			if len(frame.Inputs) == 0 {
				frame.SetInputsRaw(m.inputsRaw...)
			} else {
				frame.AppendInputsRaw(m.inputsRaw...)
			}
		}
		if len(m.inputsPrompt) == 0 && len(m.inputsRaw) == 0 {
			frame.SetInputsPrompt(line)
		} else {
			frame.AppendInputsPrompt(line)
		}
	}
	return frame.Render()
}

// PromptInputInline collects a single inline value with an optional default and validation.
// Empty input accepts the default. Validation errors are shown inline and reprompted.
func PromptInputInline(label, defaultValue string, validate func(string) error, theme Theme, useColor bool) (string, error) {
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

type inputInlineModel struct {
	title        string
	label        string
	defaultValue string
	validate     func(string) error
	theme        Theme
	useColor     bool
	input        textinput.Model
	value        string
	err          error
	errorLine    string
}

func newInputInlineModel(label, defaultValue string, validate func(string) error, theme Theme, useColor bool) inputInlineModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = defaultValue
	ti.Focus()
	if defaultValue != "" {
		ti.SetValue(defaultValue)
		ti.CursorEnd()
	}
	if useColor {
		ti.PlaceholderStyle = theme.Muted
	}
	return inputInlineModel{
		title:        "Inputs",
		label:        label,
		defaultValue: defaultValue,
		validate:     validate,
		theme:        theme,
		useColor:     useColor,
		input:        ti,
	}
}

func (m inputInlineModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputInlineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyEnter:
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				value = m.defaultValue
			}
			if m.validate != nil {
				if err := m.validate(value); err != nil {
					m.errorLine = err.Error()
					return m, nil
				}
			}
			m.value = value
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if strings.TrimSpace(m.input.Value()) != "" {
		m.errorLine = ""
	}
	return m, cmd
}

func (m inputInlineModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, m.label)
	defaultText := ""
	if strings.TrimSpace(m.defaultValue) != "" {
		defaultText = fmt.Sprintf(" [default: %s]", m.defaultValue)
	}
	line := fmt.Sprintf("%s%s: %s", label, defaultText, m.input.View())
	frame.SetInputsPrompt(line)
	if strings.TrimSpace(m.errorLine) != "" {
		errLine := m.errorLine
		if m.useColor {
			errLine = m.theme.Error.Render(errLine)
		}
		frame.AppendInputsRaw(fmt.Sprintf("%s%s %s", output.Indent+output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), errLine))
	}
	return frame.Render()
}

// PromptTemplateRepos lets users pick one or more repos from a list with filtering.
// It can also collect a template name when not provided.
func PromptTemplateRepos(title string, templateName string, choices []PromptChoice, theme Theme, useColor bool) (string, []string, error) {
	model := newTemplateRepoSelectModel(title, templateName, choices, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", nil, err
	}
	final := out.(templateRepoSelectModel)
	if final.err != nil {
		return "", nil, final.err
	}
	return strings.TrimSpace(final.templateName), append([]string(nil), final.selected...), nil
}

// PromptTemplateName asks for a template name via text input.
func PromptTemplateName(title string, defaultValue string, theme Theme, useColor bool) (string, error) {
	model := newTemplateNameModel(title, defaultValue, theme, useColor)
	out, err := runProgram(model)
	if err != nil {
		return "", err
	}
	final := out.(templateNameModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.value), nil
}

// PromptChoiceSelect lets users pick a single choice from a list with filtering.
func PromptChoiceSelect(title, label string, choices []PromptChoice, theme Theme, useColor bool) (string, error) {
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

func PromptCreateFlow(title string, startMode string, defaultWorkspaceID string, templateName string, templates []string, templateErr error, repoChoices []PromptChoice, repoErr error, reviewRepos []PromptChoice, issueRepos []PromptChoice, loadReview func(string) ([]PromptChoice, error), loadIssue func(string) ([]PromptChoice, error), loadTemplateRepos func(string) ([]string, error), validateBranch func(string) error, theme Theme, useColor bool, selectedRepo string) (string, string, string, string, []string, string, []string, string, []IssueSelection, string, error) {
	model := newCreateFlowModel(title, templates, templateErr, repoChoices, repoErr, defaultWorkspaceID, templateName, reviewRepos, issueRepos, loadReview, loadIssue, loadTemplateRepos, validateBranch, theme, useColor, startMode, selectedRepo)
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
	return final.mode, final.templateName(), final.workspaceID(), final.description, append([]string(nil), final.branches...), final.reviewRepo, append([]string(nil), final.reviewPRs...), final.issueRepo, append([]IssueSelection(nil), final.issueIssues...), final.repoSelected, nil
}

type templateRepoSelectModel struct {
	title        string
	templateName string
	choices      []PromptChoice
	filtered     []PromptChoice
	selected     []string
	addedNote    string

	theme    Theme
	useColor bool

	nameInput textinput.Model
	repoInput textinput.Model

	stage     templateRepoStage
	cursor    int
	err       error
	errorLine string

	height int
}

type templateRepoStage int

const (
	stageTemplateName templateRepoStage = iota
	stageRepoSelect
)

func newTemplateRepoSelectModel(title string, templateName string, choices []PromptChoice, theme Theme, useColor bool) templateRepoSelectModel {
	repoInput := textinput.New()
	repoInput.Prompt = ""
	repoInput.Placeholder = "search"
	if useColor {
		repoInput.PlaceholderStyle = theme.Muted
	}

	nameInput := textinput.New()
	nameInput.Prompt = ""
	nameInput.Placeholder = "template name"
	nameInput.SetValue(templateName)
	if useColor {
		nameInput.PlaceholderStyle = theme.Muted
	}

	stage := stageRepoSelect
	if strings.TrimSpace(templateName) == "" {
		stage = stageTemplateName
		nameInput.Focus()
	} else {
		repoInput.Focus()
	}

	m := templateRepoSelectModel{
		title:        title,
		templateName: templateName,
		choices:      choices,
		theme:        theme,
		useColor:     useColor,
		nameInput:    nameInput,
		repoInput:    repoInput,
		stage:        stage,
	}
	m.filtered = m.filterChoices()
	return m
}

func (m templateRepoSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m templateRepoSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyCtrlD:
			if m.stage == stageTemplateName {
				if strings.TrimSpace(m.nameInput.Value()) == "" {
					m.errorLine = "template name is required"
					return m, nil
				}
				m.templateName = strings.TrimSpace(m.nameInput.Value())
				m.stage = stageRepoSelect
				m.repoInput.Focus()
				m.errorLine = ""
				return m, nil
			}
			if len(m.selected) == 0 {
				m.errorLine = "select at least one repo"
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if m.stage == stageTemplateName {
				value := strings.TrimSpace(m.nameInput.Value())
				if value == "" {
					m.errorLine = "template name is required"
					return m, nil
				}
				m.templateName = value
				m.stage = stageRepoSelect
				m.repoInput.Focus()
				m.errorLine = ""
				return m, nil
			}
			value := strings.TrimSpace(m.repoInput.Value())
			if value == "done" {
				if len(m.selected) == 0 {
					m.errorLine = "select at least one repo"
					return m, nil
				}
				return m, tea.Quit
			}
			if len(m.filtered) == 0 {
				return m, nil
			}
			choice := m.filtered[m.cursor]
			m.selected = append(m.selected, choice.Value)
			m.choices = removeChoice(m.choices, choice.Value)
			m.repoInput.SetValue("")
			m.filtered = m.filterChoices()
			if m.cursor >= len(m.filtered) {
				m.cursor = max(0, len(m.filtered)-1)
			}
			m.addedNote = choice.Label
			m.errorLine = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.stage == stageTemplateName {
		m.nameInput, cmd = m.nameInput.Update(msg)
		if strings.TrimSpace(m.nameInput.Value()) != "" {
			m.errorLine = ""
		}
	} else {
		m.repoInput, cmd = m.repoInput.Update(msg)
		m.filtered = m.filterChoices()
		if m.cursor >= len(m.filtered) {
			m.cursor = max(0, len(m.filtered)-1)
		}
	}
	return m, cmd
}

func (m templateRepoSelectModel) View() string {
	frame := NewFrame(m.theme, m.useColor)

	labelName := promptLabel(m.theme, m.useColor, "template name")
	templateName := m.templateName
	if m.stage == stageTemplateName {
		templateName = m.nameInput.View()
	}
	labelRepo := promptLabel(m.theme, m.useColor, "repo")
	repoInput := m.repoInput.View()
	if m.stage == stageTemplateName {
		repoInput = ""
	}
	frame.SetInputsPrompt(
		fmt.Sprintf("%s: %s", labelName, templateName),
		fmt.Sprintf("%s: %s", labelRepo, repoInput),
	)

	selectedLines := collectLines(func(b *strings.Builder) {
		renderSelectedRepoTree(b, m.selected, m.useColor, m.theme)
	})
	infoLines := 1 + len(selectedLines) + 1
	if m.errorLine != "" {
		infoLines++
	}
	maxLines := listMaxLines(m.height, 2, infoLines)
	rawLines := collectLines(func(b *strings.Builder) {
		renderRepoChoiceList(b, m.filtered, m.cursor, maxLines, m.useColor, m.theme)
	})
	frame.AppendInputsRaw(rawLines...)

	if m.useColor {
		frame.SetInfo(m.theme.Accent.Render("selected"))
	} else {
		frame.SetInfo("selected")
	}
	frame.AppendInfoRaw(selectedLines...)

	if m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}

	infoPrefix := mutedToken(m.theme, m.useColor, output.StepPrefix)
	frame.AppendInfoRaw(
		fmt.Sprintf("%s%s finish: Ctrl+D or type \"done\"", output.Indent, infoPrefix),
	)
	return frame.Render()
}

func (m templateRepoSelectModel) filterChoices() []PromptChoice {
	q := strings.ToLower(strings.TrimSpace(m.repoInput.Value()))
	if q == "" {
		return append([]PromptChoice(nil), m.choices...)
	}
	var out []PromptChoice
	for _, item := range m.choices {
		if strings.Contains(strings.ToLower(item.Label), q) || strings.Contains(strings.ToLower(item.Value), q) {
			out = append(out, item)
		}
	}
	return out
}

func removeChoice(items []PromptChoice, value string) []PromptChoice {
	var out []PromptChoice
	for _, item := range items {
		if item.Value == value {
			continue
		}
		out = append(out, item)
	}
	return out
}

func removeWorkspaceChoice(items []WorkspaceChoice, workspaceID string) []WorkspaceChoice {
	var out []WorkspaceChoice
	for _, item := range items {
		if item.ID == workspaceID {
			continue
		}
		out = append(out, item)
	}
	return out
}

type choiceSelectModel struct {
	title     string
	label     string
	choices   []PromptChoice
	filtered  []PromptChoice
	cursor    int
	value     string
	err       error
	errorLine string
	done      bool

	theme    Theme
	useColor bool
	input    textinput.Model

	height int
}

func newChoiceSelectModel(title, label string, choices []PromptChoice, theme Theme, useColor bool) choiceSelectModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := choiceSelectModel{
		title:    title,
		label:    label,
		choices:  choices,
		theme:    theme,
		useColor: useColor,
		input:    input,
	}
	m.filtered = m.filterChoices()
	return m
}

func (m choiceSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m choiceSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if len(m.filtered) == 0 {
				m.errorLine = "select a value"
				return m, nil
			}
			choice := m.filtered[m.cursor]
			m.value = choice.Value
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = m.filterChoices()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	if strings.TrimSpace(m.input.Value()) != "" {
		m.errorLine = ""
	}
	return m, cmd
}

func (m choiceSelectModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, m.label)
	frame.SetInputsPrompt(fmt.Sprintf("%s: %s", label, m.input.View()))

	infoLines := 0
	if m.errorLine != "" {
		infoLines = 1
	}
	maxLines := listMaxLines(m.height, 1, infoLines)
	rawLines := collectLines(func(b *strings.Builder) {
		renderRepoChoiceList(b, m.filtered, m.cursor, maxLines, m.useColor, m.theme)
	})
	frame.AppendInputsRaw(rawLines...)

	if m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}
	return frame.Render()
}

func (m choiceSelectModel) filterChoices() []PromptChoice {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]PromptChoice(nil), m.choices...)
	}
	var out []PromptChoice
	for _, item := range m.choices {
		if strings.Contains(strings.ToLower(item.Label), q) || strings.Contains(strings.ToLower(item.Value), q) {
			out = append(out, item)
		}
	}
	return out
}

type multiSelectModel struct {
	title          string
	label          string
	choices        []PromptChoice
	filtered       []PromptChoice
	selected       []PromptChoice
	selectedValues []string
	cursor         int
	err            error
	errorLine      string
	done           bool

	theme    Theme
	useColor bool
	input    textinput.Model

	height int
}

func newMultiSelectModel(title, label string, choices []PromptChoice, theme Theme, useColor bool) multiSelectModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := multiSelectModel{
		title:    title,
		label:    label,
		choices:  choices,
		theme:    theme,
		useColor: useColor,
		input:    input,
	}
	m.filtered = m.filterChoices()
	return m
}

func (m multiSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyCtrlD:
			if len(m.selected) == 0 {
				m.errorLine = fmt.Sprintf("select at least one %s", m.label)
				return m, nil
			}
			m.done = true
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			value := strings.TrimSpace(m.input.Value())
			if value == "done" {
				if len(m.selected) == 0 {
					m.errorLine = fmt.Sprintf("select at least one %s", m.label)
					return m, nil
				}
				m.done = true
				return m, tea.Quit
			}
			if len(m.filtered) == 0 {
				return m, nil
			}
			choice := m.filtered[m.cursor]
			m.selected = append(m.selected, choice)
			m.selectedValues = append(m.selectedValues, choice.Value)
			m.choices = removeChoice(m.choices, choice.Value)
			m.input.SetValue("")
			m.filtered = m.filterChoices()
			if m.cursor >= len(m.filtered) {
				m.cursor = max(0, len(m.filtered)-1)
			}
			m.errorLine = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = m.filterChoices()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m multiSelectModel) View() string {
	return renderMultiSelectFrame(m, m.height)
}

func renderMultiSelectFrame(model multiSelectModel, height int, headerLines ...string) string {
	frame := NewFrame(model.theme, model.useColor)
	lines := append([]string(nil), headerLines...)
	label := promptLabel(model.theme, model.useColor, model.label)
	lines = append(lines, fmt.Sprintf("%s: %s", label, model.input.View()))
	frame.SetInputsPrompt(lines...)

	selectedLines := collectLines(func(b *strings.Builder) {
		renderSelectedChoiceTree(b, model.selected, model.useColor, model.theme)
	})
	infoLines := 1 + len(selectedLines) + 1
	if model.errorLine != "" {
		infoLines++
	}
	maxLines := listMaxLines(height, len(lines), infoLines)
	rawLines := collectLines(func(b *strings.Builder) {
		renderRepoChoiceList(b, model.filtered, model.cursor, maxLines, model.useColor, model.theme)
	})
	frame.AppendInputsRaw(rawLines...)

	if model.useColor {
		frame.SetInfo(model.theme.Accent.Render("selected"))
	} else {
		frame.SetInfo("selected")
	}
	frame.AppendInfoRaw(selectedLines...)

	if model.errorLine != "" {
		msg := model.errorLine
		if model.useColor {
			msg = model.theme.Error.Render(msg)
		}
		frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(model.theme, model.useColor, output.LogConnector), msg))
	}

	infoPrefix := mutedToken(model.theme, model.useColor, output.StepPrefix)
	frame.AppendInfoRaw(
		fmt.Sprintf("%s%s finish: Ctrl+D or type \"done\"", output.Indent, infoPrefix),
	)
	return frame.Render()
}

func renderIssueBranchEditFrame(model issueBranchSelectModel, headerLines ...string) string {
	frame := NewFrame(model.theme, model.useColor)
	lines := append([]string(nil), headerLines...)
	label := promptLabel(model.theme, model.useColor, model.label)
	lines = append(lines, fmt.Sprintf("%s: edit branches", label))
	frame.SetInputsPrompt(lines...)

	infoLines := 1
	if model.errorLine != "" {
		infoLines++
	}
	maxLines := listMaxLines(model.height, len(lines), infoLines)
	rawLines := collectLines(func(b *strings.Builder) {
		if len(model.selected) == 0 {
			msg := "no selections"
			if model.useColor {
				msg = model.theme.Muted.Render(msg)
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(model.theme, model.useColor, output.LogConnector), msg))
			return
		}
		start, end := listWindow(len(model.selected), model.branchCursor, maxLines)
		for i := start; i < end; i++ {
			choice := model.selected[i]
			display := choice.Label
			branchValue := strings.TrimSpace(model.branchInputs[i].Value())
			if branchValue == "" {
				branchValue = defaultIssueBranch(choice.Value)
			}
			branchDisplay := branchValue
			if i == model.branchCursor {
				if model.useColor {
					display = lipgloss.NewStyle().Bold(true).Render(display)
				}
				branchDisplay = model.branchInputs[i].View()
			}
			b.WriteString(fmt.Sprintf("%s%s %s | branch: %s\n", output.Indent+output.Indent, mutedToken(model.theme, model.useColor, output.LogConnector), display, branchDisplay))
		}
	})
	frame.AppendInputsRaw(rawLines...)

	if model.errorLine != "" {
		msg := model.errorLine
		if model.useColor {
			msg = model.theme.Error.Render(msg)
		}
		frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(model.theme, model.useColor, output.LogConnector), msg))
	}

	infoPrefix := mutedToken(model.theme, model.useColor, output.StepPrefix)
	frame.AppendInfoRaw(
		fmt.Sprintf("%s%s finish: Ctrl+D", output.Indent, infoPrefix),
	)
	return frame.Render()
}

func (m multiSelectModel) filterChoices() []PromptChoice {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]PromptChoice(nil), m.choices...)
	}
	var out []PromptChoice
	for _, item := range m.choices {
		if strings.Contains(strings.ToLower(item.Label), q) || strings.Contains(strings.ToLower(item.Value), q) {
			out = append(out, item)
		}
	}
	return out
}

type issueBranchStage int

const (
	issueBranchStageSelect issueBranchStage = iota
	issueBranchStageEdit
)

type issueBranchSelectModel struct {
	title          string
	label          string
	choices        []PromptChoice
	filtered       []PromptChoice
	selected       []PromptChoice
	cursor         int
	err            error
	errorLine      string
	done           bool
	stage          issueBranchStage
	branchCursor   int
	branchInputs   []textinput.Model
	selectedIssues []IssueSelection

	theme          Theme
	useColor       bool
	input          textinput.Model
	validateBranch func(string) error

	height int
}

func newIssueBranchSelectModel(title, label string, choices []PromptChoice, validateBranch func(string) error, theme Theme, useColor bool) issueBranchSelectModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := issueBranchSelectModel{
		title:          title,
		label:          label,
		choices:        choices,
		theme:          theme,
		useColor:       useColor,
		input:          input,
		validateBranch: validateBranch,
	}
	m.filtered = m.filterChoices()
	return m
}

func (m issueBranchSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m issueBranchSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyUp:
			if m.stage == issueBranchStageSelect && m.cursor > 0 {
				m.cursor--
				return m, nil
			}
			if m.stage == issueBranchStageEdit && m.branchCursor > 0 {
				m.branchCursor--
				m = m.focusBranchInput(m.branchCursor)
				return m, nil
			}
		case tea.KeyDown:
			if m.stage == issueBranchStageSelect && m.cursor < len(m.filtered)-1 {
				m.cursor++
				return m, nil
			}
			if m.stage == issueBranchStageEdit && m.branchCursor < len(m.branchInputs)-1 {
				m.branchCursor++
				m = m.focusBranchInput(m.branchCursor)
				return m, nil
			}
		case tea.KeyCtrlD:
			if m.stage == issueBranchStageSelect {
				if len(m.selected) == 0 {
					m.errorLine = fmt.Sprintf("select at least one %s", m.label)
					return m, nil
				}
				m = m.startBranchEdit()
				return m, nil
			}
			if m.stage == issueBranchStageEdit {
				var ok bool
				m, ok = m.finalizeBranches()
				if ok {
					return m, tea.Quit
				}
				return m, nil
			}
		case tea.KeyEnter:
			if m.stage == issueBranchStageSelect {
				value := strings.TrimSpace(m.input.Value())
				if value == "done" {
					if len(m.selected) == 0 {
						m.errorLine = fmt.Sprintf("select at least one %s", m.label)
						return m, nil
					}
					m = m.startBranchEdit()
					return m, nil
				}
				if len(m.filtered) == 0 {
					return m, nil
				}
				choice := m.filtered[m.cursor]
				m.selected = append(m.selected, choice)
				m.choices = removeChoice(m.choices, choice.Value)
				m.input.SetValue("")
				m.filtered = m.filterChoices()
				if m.cursor >= len(m.filtered) {
					m.cursor = max(0, len(m.filtered)-1)
				}
				m.errorLine = ""
				return m, nil
			}
			if m.stage == issueBranchStageEdit {
				if len(m.branchInputs) == 0 {
					return m, nil
				}
				branch := strings.TrimSpace(m.branchInputs[m.branchCursor].Value())
				if branch == "" {
					branch = defaultIssueBranch(m.selected[m.branchCursor].Value)
					m.branchInputs[m.branchCursor].SetValue(branch)
					m.branchInputs[m.branchCursor].CursorEnd()
				}
				if m.validateBranch != nil {
					if err := m.validateBranch(branch); err != nil {
						m.errorLine = err.Error()
						return m, nil
					}
				}
				m.errorLine = ""
				if m.branchCursor < len(m.branchInputs)-1 {
					m.branchCursor++
					m = m.focusBranchInput(m.branchCursor)
				}
				return m, nil
			}
		}
	}

	if m.stage == issueBranchStageSelect {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.filtered = m.filterChoices()
		if m.cursor >= len(m.filtered) {
			m.cursor = max(0, len(m.filtered)-1)
		}
		return m, cmd
	}

	if m.stage == issueBranchStageEdit && len(m.branchInputs) > 0 {
		var cmd tea.Cmd
		m.branchInputs[m.branchCursor], cmd = m.branchInputs[m.branchCursor].Update(msg)
		if strings.TrimSpace(m.branchInputs[m.branchCursor].Value()) != "" {
			m.errorLine = ""
		}
		return m, cmd
	}

	return m, nil
}

func (m issueBranchSelectModel) View() string {
	if m.stage == issueBranchStageEdit {
		return renderIssueBranchEditFrame(m)
	}
	return renderMultiSelectFrame(multiSelectModel{
		title:     m.title,
		label:     m.label,
		choices:   m.choices,
		filtered:  m.filtered,
		selected:  m.selected,
		cursor:    m.cursor,
		errorLine: m.errorLine,
		theme:     m.theme,
		useColor:  m.useColor,
		input:     m.input,
		height:    m.height,
	}, m.height)
}

func (m issueBranchSelectModel) filterChoices() []PromptChoice {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]PromptChoice(nil), m.choices...)
	}
	var out []PromptChoice
	for _, item := range m.choices {
		if strings.Contains(strings.ToLower(item.Label), q) || strings.Contains(strings.ToLower(item.Value), q) {
			out = append(out, item)
		}
	}
	return out
}

func (m issueBranchSelectModel) startBranchEdit() issueBranchSelectModel {
	m.stage = issueBranchStageEdit
	m.branchInputs = make([]textinput.Model, len(m.selected))
	for i, choice := range m.selected {
		input := textinput.New()
		input.Prompt = ""
		input.Placeholder = "branch"
		input.SetValue(defaultIssueBranch(choice.Value))
		if m.useColor {
			input.PlaceholderStyle = m.theme.Muted
		}
		m.branchInputs[i] = input
	}
	m.branchCursor = 0
	m = m.focusBranchInput(0)
	m.errorLine = ""
	return m
}

func (m issueBranchSelectModel) focusBranchInput(index int) issueBranchSelectModel {
	for i := range m.branchInputs {
		if i == index {
			m.branchInputs[i].Focus()
		} else {
			m.branchInputs[i].Blur()
		}
	}
	return m
}

func (m issueBranchSelectModel) finalizeBranches() (issueBranchSelectModel, bool) {
	branchByValue := make(map[string]int, len(m.selected))
	m.selectedIssues = make([]IssueSelection, len(m.selected))
	for i, choice := range m.selected {
		branch := strings.TrimSpace(m.branchInputs[i].Value())
		if branch == "" {
			branch = defaultIssueBranch(choice.Value)
			m.branchInputs[i].SetValue(branch)
			m.branchInputs[i].CursorEnd()
		}
		if m.validateBranch != nil {
			if err := m.validateBranch(branch); err != nil {
				m.errorLine = fmt.Sprintf("%s: %s", choice.Label, err.Error())
				m.branchCursor = i
				m = m.focusBranchInput(i)
				return m, false
			}
		}
		if prev, exists := branchByValue[branch]; exists {
			m.errorLine = fmt.Sprintf("branch %q already used for %s; re-enter", branch, m.selected[prev].Label)
			m.branchCursor = i
			m = m.focusBranchInput(i)
			return m, false
		}
		branchByValue[branch] = i
		m.selectedIssues[i] = IssueSelection{
			Value:  choice.Value,
			Branch: branch,
		}
	}
	m.done = true
	return m, true
}

func defaultIssueBranch(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "issue/"
	}
	return fmt.Sprintf("issue/%s", value)
}

type templateNameModel struct {
	title     string
	theme     Theme
	useColor  bool
	input     textinput.Model
	value     string
	err       error
	errorLine string
}

func newTemplateNameModel(title string, defaultValue string, theme Theme, useColor bool) templateNameModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "template name"
	input.SetValue(defaultValue)
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	return templateNameModel{
		title:    title,
		theme:    theme,
		useColor: useColor,
		input:    input,
	}
}

func (m templateNameModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m templateNameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyEnter:
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				m.errorLine = "required"
				return m, nil
			}
			m.value = value
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if strings.TrimSpace(m.input.Value()) != "" {
		m.errorLine = ""
	}
	return m, cmd
}

func (m templateNameModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "template name")
	frame.SetInputsPrompt(fmt.Sprintf("%s: %s", label, m.input.View()))
	if m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		frame.AppendInputsRaw(fmt.Sprintf("%s%s %s", output.Indent+output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}
	return frame.Render()
}

func promptLabel(theme Theme, useColor bool, label string) string {
	if useColor {
		return theme.Accent.Render(label)
	}
	return label
}

func mutedToken(theme Theme, useColor bool, token string) string {
	if useColor {
		return theme.Muted.Render(token)
	}
	return token
}

func runProgram(model tea.Model) (tea.Model, error) {
	return tea.NewProgram(model).Run()
}

func runProgramWithOutput(model tea.Model, out io.Writer) (tea.Model, error) {
	return tea.NewProgram(model, tea.WithOutput(out)).Run()
}

func collectLines(render func(*strings.Builder)) []string {
	var b strings.Builder
	render(&b)
	return splitLines(b.String())
}

func splitLines(text string) []string {
	trimmed := strings.TrimRight(text, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return nil
	}
	parts := strings.Split(trimmed, "\n")
	out := make([]string, 0, len(parts))
	for _, line := range parts {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const defaultViewportHeight = 20

func listMaxLines(height int, inputLines int, infoLines int) int {
	if height <= 0 {
		height = defaultViewportHeight
	}
	total := 0
	if inputLines > 0 {
		total += 1 + inputLines + 1
	}
	if infoLines > 0 {
		total += 1 + infoLines + 1
	}
	maxLines := height - total
	if maxLines < 1 {
		return 1
	}
	return maxLines
}

type workspaceSelectModel struct {
	title      string
	workspaces []WorkspaceChoice
	blocked    []BlockedChoice
	theme      Theme
	useColor   bool

	input    textinput.Model
	filtered []WorkspaceChoice
	cursor   int
	err      error

	workspaceID string

	height int
}

func newWorkspaceSelectModel(title string, workspaces []WorkspaceChoice, theme Theme, useColor bool) workspaceSelectModel {
	return newWorkspaceSelectModelWithBlocked(title, workspaces, nil, theme, useColor)
}

func newWorkspaceSelectModelWithBlocked(title string, workspaces []WorkspaceChoice, blocked []BlockedChoice, theme Theme, useColor bool) workspaceSelectModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := workspaceSelectModel{
		title:      title,
		workspaces: workspaces,
		blocked:    blocked,
		theme:      theme,
		useColor:   useColor,
		input:      input,
	}
	m.filtered = m.filterWorkspaces()
	return m
}

func (m workspaceSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m workspaceSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if len(m.filtered) == 0 {
				return m, nil
			}
			m.workspaceID = m.filtered[m.cursor].ID
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = m.filterWorkspaces()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m workspaceSelectModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "workspace id")
	frame.SetInputsPrompt(fmt.Sprintf("%s: %s", label, m.input.View()))
	var blockedLines []string
	infoLines := 0
	if len(m.blocked) > 0 {
		blockedLines = collectLines(func(b *strings.Builder) {
			renderBlockedChoiceList(b, m.blocked, m.useColor, m.theme)
		})
		infoLines = 1 + len(blockedLines)
	}
	maxLines := listMaxLines(m.height, 1, infoLines)
	rawLines := collectLines(func(b *strings.Builder) {
		renderWorkspaceChoiceList(b, m.filtered, m.cursor, maxLines, m.useColor, m.theme)
	})
	frame.AppendInputsRaw(rawLines...)
	if len(m.blocked) > 0 {
		frame.SetInfo("blocked workspaces")
		frame.AppendInfoRaw(blockedLines...)
	}
	return frame.Render()
}

func (m workspaceSelectModel) filterWorkspaces() []WorkspaceChoice {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]WorkspaceChoice(nil), m.workspaces...)
	}
	var out []WorkspaceChoice
	for _, item := range m.workspaces {
		if strings.Contains(strings.ToLower(item.ID), q) || strings.Contains(strings.ToLower(item.Description), q) {
			out = append(out, item)
		}
	}
	return out
}

type multiSelectStage int

const (
	multiSelectStageSelect multiSelectStage = iota
	multiSelectStageConfirm
)

type workspaceMultiSelectModel struct {
	title            string
	workspaces       []WorkspaceChoice
	blocked          []BlockedChoice
	filtered         []WorkspaceChoice
	selected         []WorkspaceChoice
	selectedIDs      []string
	cursor           int
	err              error
	errorLine        string
	canceled         bool
	stage            multiSelectStage
	confirmModel     confirmInlineModel
	confirmInputsRaw []string

	theme    Theme
	useColor bool
	input    textinput.Model

	height int
}

func newWorkspaceMultiSelectModel(title string, workspaces []WorkspaceChoice, blocked []BlockedChoice, theme Theme, useColor bool) workspaceMultiSelectModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := workspaceMultiSelectModel{
		title:      title,
		workspaces: workspaces,
		blocked:    blocked,
		theme:      theme,
		useColor:   useColor,
		input:      input,
	}
	m.filtered = m.filterWorkspaces()
	return m
}

func (m workspaceMultiSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m workspaceMultiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = size.Height
		return m, nil
	}
	if m.stage == multiSelectStageConfirm {
		model, _ := m.confirmModel.Update(msg)
		m.confirmModel = model.(confirmInlineModel)
		if m.confirmModel.err != nil {
			if errors.Is(m.confirmModel.err, ErrPromptCanceled) {
				m.canceled = true
			}
			return m, tea.Quit
		}
		if m.confirmModel.done {
			if !m.confirmModel.value {
				m.canceled = true
			}
			return m, tea.Quit
		}
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyCtrlD:
			if len(m.selected) == 0 {
				m.errorLine = "select at least one workspace"
				return m, nil
			}
			return m.startConfirmIfNeeded()
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			value := strings.TrimSpace(m.input.Value())
			if value == "done" {
				if len(m.selected) == 0 {
					m.errorLine = "select at least one workspace"
					return m, nil
				}
				return m.startConfirmIfNeeded()
			}
			if len(m.filtered) == 0 {
				return m, nil
			}
			choice := m.filtered[m.cursor]
			m.selected = append(m.selected, choice)
			m.selectedIDs = append(m.selectedIDs, choice.ID)
			m.workspaces = removeWorkspaceChoice(m.workspaces, choice.ID)
			m.input.SetValue("")
			m.filtered = m.filterWorkspaces()
			if m.cursor >= len(m.filtered) {
				m.cursor = max(0, len(m.filtered)-1)
			}
			m.errorLine = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = m.filterWorkspaces()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return m, cmd
}

func (m workspaceMultiSelectModel) View() string {
	if m.stage == multiSelectStageConfirm {
		frame := NewFrame(m.theme, m.useColor)
		label := promptLabel(m.theme, m.useColor, m.confirmModel.label)
		line := fmt.Sprintf("%s (y/n): %s", label, m.confirmModel.input.View())
		frame.SetInputsPrompt(line)
		if len(m.confirmInputsRaw) > 0 {
			frame.AppendInputsRaw(m.confirmInputsRaw...)
		}
		return frame.Render()
	}
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "workspace")
	frame.SetInputsPrompt(fmt.Sprintf("%s: %s", label, m.input.View()))
	selectedLines := collectLines(func(b *strings.Builder) {
		renderSelectedWorkspaceTree(b, m.selected, m.useColor, m.theme)
	})
	var blockedLines []string
	infoLines := 1 + len(selectedLines) + 1
	if m.errorLine != "" {
		infoLines++
	}
	if len(m.blocked) > 0 {
		blockedLines = collectLines(func(b *strings.Builder) {
			renderBlockedChoiceList(b, m.blocked, m.useColor, m.theme)
		})
		infoLines += 1 + len(blockedLines)
	}
	maxLines := listMaxLines(m.height, 1, infoLines)
	rawLines := collectLines(func(b *strings.Builder) {
		renderWorkspaceChoiceList(b, m.filtered, m.cursor, maxLines, m.useColor, m.theme)
	})
	frame.AppendInputsRaw(rawLines...)

	if m.useColor {
		frame.SetInfo(m.theme.Accent.Render("selected"))
	} else {
		frame.SetInfo("selected")
	}
	frame.AppendInfoRaw(selectedLines...)

	if m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}

	infoPrefix := mutedToken(m.theme, m.useColor, output.StepPrefix)
	frame.AppendInfoRaw(
		fmt.Sprintf("%s%s finish: Ctrl+D or type \"done\"", output.Indent, infoPrefix),
	)

	if len(m.blocked) > 0 {
		frame.AppendInfo("blocked workspaces")
		frame.AppendInfoRaw(blockedLines...)
	}
	return frame.Render()
}

func (m workspaceMultiSelectModel) startConfirmIfNeeded() (workspaceMultiSelectModel, tea.Cmd) {
	label, needConfirm := confirmLabelForSelection(m.selected)
	if !needConfirm {
		return m, tea.Quit
	}
	m.confirmModel = newConfirmInlineModel(label, m.theme, m.useColor, false, nil, nil)
	m.confirmInputsRaw = WorkspaceChoiceLines(m.selected, -1, m.useColor, m.theme)
	m.stage = multiSelectStageConfirm
	return m, nil
}

func confirmLabelForSelection(selected []WorkspaceChoice) (string, bool) {
	if len(selected) == 0 {
		return "", false
	}
	if len(selected) == 1 {
		warn := strings.TrimSpace(selected[0].Warning)
		if warn == "" {
			return "", false
		}
		switch strings.ToLower(warn) {
		case "dirty changes":
			return "This workspace has uncommitted changes. Remove anyway?", true
		case "unpushed commits":
			return "This workspace has unpushed commits. Remove anyway?", true
		case "diverged or upstream missing":
			return "This workspace has diverged from upstream. Remove anyway?", true
		case "status unknown":
			return "Workspace status could not be read. Remove anyway?", true
		default:
			return "This workspace has warnings. Remove anyway?", true
		}
	}
	hasWarning := false
	hasStrong := false
	for _, item := range selected {
		if strings.TrimSpace(item.Warning) != "" {
			hasWarning = true
		}
		if item.WarningStrong {
			hasStrong = true
		}
	}
	if !hasWarning {
		return fmt.Sprintf("Remove %d workspaces?", len(selected)), true
	}
	if hasStrong {
		return fmt.Sprintf("Selected workspaces include uncommitted changes or status errors. Remove %d workspaces anyway?", len(selected)), true
	}
	return fmt.Sprintf("Selected workspaces have warnings. Remove %d workspaces anyway?", len(selected)), true
}

func (m workspaceMultiSelectModel) filterWorkspaces() []WorkspaceChoice {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]WorkspaceChoice(nil), m.workspaces...)
	}
	var out []WorkspaceChoice
	for _, item := range m.workspaces {
		if strings.Contains(strings.ToLower(item.ID), q) || strings.Contains(strings.ToLower(item.Description), q) {
			out = append(out, item)
			continue
		}
		for _, repo := range item.Repos {
			if strings.Contains(strings.ToLower(repo.Label), q) {
				out = append(out, item)
				break
			}
		}
	}
	return out
}

type addInputsStage int

const (
	addStageWorkspace addInputsStage = iota
	addStageRepo
	addStageDone
)

type addInputsModel struct {
	title       string
	workspaces  []WorkspaceChoice
	repos       []PromptChoice
	workspaceID string
	repoSpec    string
	repoLabel   string

	stage    addInputsStage
	theme    Theme
	useColor bool

	wsInput   textinput.Model
	repoInput textinput.Model

	wsFiltered   []WorkspaceChoice
	repoFiltered []PromptChoice
	cursor       int
	wsRepoKeys   map[string]map[string]struct{}

	err error

	height int
}

func newAddInputsModel(title string, workspaces []WorkspaceChoice, repos []PromptChoice, workspaceID, repoSpec string, theme Theme, useColor bool) addInputsModel {
	wsInput := textinput.New()
	wsInput.Prompt = ""
	wsInput.Placeholder = "search"
	if useColor {
		wsInput.PlaceholderStyle = theme.Muted
	}

	repoInput := textinput.New()
	repoInput.Prompt = ""
	repoInput.Placeholder = "search"
	if useColor {
		repoInput.PlaceholderStyle = theme.Muted
	}

	stage := addStageWorkspace
	if strings.TrimSpace(workspaceID) != "" {
		stage = addStageRepo
		repoInput.Focus()
	} else {
		wsInput.Focus()
	}

	m := addInputsModel{
		title:       title,
		workspaces:  workspaces,
		repos:       repos,
		workspaceID: workspaceID,
		repoSpec:    repoSpec,
		stage:       stage,
		theme:       theme,
		useColor:    useColor,
		wsInput:     wsInput,
		repoInput:   repoInput,
		wsRepoKeys:  buildWorkspaceRepoKeys(workspaces),
	}
	m.wsFiltered = m.filterWorkspaces()
	m.repoFiltered = m.filterRepos()
	return m
}

func (m addInputsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m addInputsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < m.maxCursor() {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if m.stage == addStageWorkspace {
				if len(m.wsFiltered) == 0 {
					return m, nil
				}
				m.workspaceID = m.wsFiltered[m.cursor].ID
				m.stage = addStageRepo
				m.cursor = 0
				m.repoInput.Focus()
				return m, nil
			}
			if m.stage == addStageRepo {
				if len(m.repoFiltered) == 0 {
					return m, nil
				}
				choice := m.repoFiltered[m.cursor]
				m.repoSpec = choice.Value
				m.repoLabel = choice.Label
				m.stage = addStageDone
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	if m.stage == addStageWorkspace {
		m.wsInput, cmd = m.wsInput.Update(msg)
		m.wsFiltered = m.filterWorkspaces()
		if m.cursor >= len(m.wsFiltered) {
			m.cursor = max(0, len(m.wsFiltered)-1)
		}
	} else if m.stage == addStageRepo {
		m.repoInput, cmd = m.repoInput.Update(msg)
		m.repoFiltered = m.filterRepos()
		if m.cursor >= len(m.repoFiltered) {
			m.cursor = max(0, len(m.repoFiltered)-1)
		}
	}
	return m, cmd
}

func (m addInputsModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	labelWorkspace := promptLabel(m.theme, m.useColor, "workspace id")
	if m.stage == addStageWorkspace {
		frame.SetInputsPrompt(fmt.Sprintf("%s: %s", labelWorkspace, m.wsInput.View()))
		maxLines := listMaxLines(m.height, 1, 0)
		rawLines := collectLines(func(b *strings.Builder) {
			renderWorkspaceChoiceList(b, m.wsFiltered, m.cursor, maxLines, m.useColor, m.theme)
		})
		frame.AppendInputsRaw(rawLines...)
	} else {
		frame.SetInputsPrompt(fmt.Sprintf("%s: %s", labelWorkspace, m.workspaceID))
		var repoDetailLines []string
		if selected, ok := m.selectedWorkspace(); ok {
			repoDetailLines = collectLines(func(b *strings.Builder) {
				renderRepoDetailList(b, selected.Repos, output.Indent+output.Indent, m.useColor, m.theme)
			})
			frame.AppendInputsRaw(repoDetailLines...)
		}

		labelRepo := promptLabel(m.theme, m.useColor, "repo")
		if m.stage == addStageRepo {
			frame.AppendInputsPrompt(fmt.Sprintf("%s: %s", labelRepo, m.repoInput.View()))
			inputLines := 1 + len(repoDetailLines) + 1
			maxLines := listMaxLines(m.height, inputLines, 0)
			rawLines := collectLines(func(b *strings.Builder) {
				renderChoiceList(b, m.repoLabels(), m.cursor, maxLines, m.useColor, m.theme)
			})
			frame.AppendInputsRaw(rawLines...)
		} else if m.repoLabel != "" {
			frame.AppendInputsPrompt(fmt.Sprintf("%s: %s", labelRepo, m.repoLabel))
		}
	}

	return frame.Render()
}

func (m addInputsModel) maxCursor() int {
	if m.stage == addStageWorkspace {
		return max(0, len(m.wsFiltered)-1)
	}
	if m.stage == addStageRepo {
		return max(0, len(m.repoFiltered)-1)
	}
	return max(0, len(m.repoFiltered)-1)
}

func (m addInputsModel) filterWorkspaces() []WorkspaceChoice {
	q := strings.ToLower(strings.TrimSpace(m.wsInput.Value()))
	if q == "" {
		return append([]WorkspaceChoice(nil), m.workspaces...)
	}
	var out []WorkspaceChoice
	for _, item := range m.workspaces {
		if strings.Contains(strings.ToLower(item.ID), q) || strings.Contains(strings.ToLower(item.Description), q) {
			out = append(out, item)
		}
	}
	return out
}

func (m addInputsModel) filterRepos() []PromptChoice {
	q := strings.ToLower(strings.TrimSpace(m.repoInput.Value()))
	blocked := m.wsRepoKeys[m.workspaceID]
	if q == "" {
		return filterRepoChoices(m.repos, blocked, "")
	}
	return filterRepoChoices(m.repos, blocked, q)
}

func (m addInputsModel) repoLabels() []string {
	labels := make([]string, 0, len(m.repoFiltered))
	for _, item := range m.repoFiltered {
		labels = append(labels, item.Label)
	}
	return labels
}

func (m addInputsModel) currentRepoLabel() string {
	if m.repoLabel != "" {
		return m.repoLabel
	}
	if strings.TrimSpace(m.repoSpec) != "" {
		return m.repoSpec
	}
	return ""
}

func formatAddInputsHeader(title, workspaceID, repoLabel string) string {
	var parts []string
	if strings.TrimSpace(workspaceID) != "" {
		parts = append(parts, fmt.Sprintf("workspace id: %s", workspaceID))
	}
	if strings.TrimSpace(repoLabel) != "" {
		parts = append(parts, fmt.Sprintf("repo: %s", repoLabel))
	}
	if len(parts) == 0 {
		return title
	}
	return fmt.Sprintf("%s (%s)", title, strings.Join(parts, ", "))
}

func listWindow(total int, cursor int, maxVisible int) (int, int) {
	if maxVisible <= 0 || total <= maxVisible {
		return 0, total
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	start := cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > total {
		start = total - maxVisible
	}
	return start, start + maxVisible
}

func renderChoiceList(b *strings.Builder, items []string, cursor int, maxVisible int, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	start, end := listWindow(len(items), cursor, maxVisible)
	for i := start; i < end; i++ {
		item := items[i]
		display := item
		if i == cursor && useColor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), display))
	}
}

func renderRepoChoiceList(b *strings.Builder, items []PromptChoice, cursor int, maxVisible int, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	start, end := listWindow(len(items), cursor, maxVisible)
	for i := start; i < end; i++ {
		item := items[i]
		display := item.Label
		if i == cursor && useColor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), display))
	}
}

func renderSelectedRepoList(b *strings.Builder, items []string, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "none"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, mutedToken(theme, useColor, output.StepPrefix), msg))
		return
	}
	for _, item := range items {
		prefix := output.StepPrefix
		if useColor {
			prefix = theme.Accent.Render(prefix)
		}
		line := fmt.Sprintf("%s%s %s", output.Indent, prefix, item)
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderSelectedChoiceList(b *strings.Builder, items []PromptChoice, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "none"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, mutedToken(theme, useColor, output.StepPrefix), msg))
		return
	}
	for _, item := range items {
		prefix := output.StepPrefix
		if useColor {
			prefix = theme.Accent.Render(prefix)
		}
		line := fmt.Sprintf("%s%s %s", output.Indent, prefix, item.Label)
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderSelectedRepoTree(b *strings.Builder, items []string, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "none"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	for i, item := range items {
		prefix := "├─ "
		if i == len(items)-1 {
			prefix = "└─ "
		}
		line := fmt.Sprintf("%s%s%s", output.Indent, prefix, item)
		if useColor {
			line = theme.Muted.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderSelectedChoiceTree(b *strings.Builder, items []PromptChoice, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "none"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	for i, item := range items {
		prefix := "├─ "
		if i == len(items)-1 {
			prefix = "└─ "
		}
		line := fmt.Sprintf("%s%s%s", output.Indent, prefix, item.Label)
		if useColor {
			line = theme.Muted.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderRepoDetailList(b *strings.Builder, repos []PromptChoice, indent string, useColor bool, theme Theme) {
	for i, repo := range repos {
		connector := "├─"
		if i == len(repos)-1 {
			connector = "└─"
		}
		line := fmt.Sprintf("%s%s %s", indent, connector, repo.Label)
		if useColor {
			line = theme.Muted.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderSelectedWorkspaceList(b *strings.Builder, items []WorkspaceChoice, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "none"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, mutedToken(theme, useColor, output.StepPrefix), msg))
		return
	}
	for _, item := range items {
		display := item.ID
		desc := strings.TrimSpace(item.Description)
		if desc != "" {
			if useColor {
				display += theme.Muted.Render(" - " + desc)
			} else {
				display += " - " + desc
			}
		}
		prefix := output.StepPrefix
		if useColor {
			prefix = theme.Accent.Render(prefix)
		}
		line := fmt.Sprintf("%s%s %s", output.Indent, prefix, display)
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderSelectedWorkspaceTree(b *strings.Builder, items []WorkspaceChoice, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "none"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	for i, item := range items {
		label := item.ID
		if strings.TrimSpace(item.Description) != "" {
			label = fmt.Sprintf("%s - %s", item.ID, item.Description)
		}
		prefix := "├─ "
		if i == len(items)-1 {
			prefix = "└─ "
		}
		line := fmt.Sprintf("%s%s%s", output.Indent, prefix, label)
		if useColor {
			line = theme.Muted.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderWorkspaceChoiceList(b *strings.Builder, items []WorkspaceChoice, cursor int, maxVisible int, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	start, end := listWindow(len(items), cursor, maxVisible)
	for i := start; i < end; i++ {
		item := items[i]
		displayID := item.ID
		hasWarn := strings.TrimSpace(item.Warning) != ""
		warnStyle := theme.SoftWarn
		if item.WarningStrong {
			warnStyle = theme.Warn
		}
		warnTag := ""
		if hasWarn {
			warnTag = "[" + shortWarningTag(item.Warning) + "]"
		}
		if useColor {
			if hasWarn {
				if i == cursor {
					displayID = theme.Warn.Copy().Bold(true).Render(displayID)
				} else {
					displayID = warnStyle.Render(displayID)
				}
			} else if i == cursor {
				displayID = lipgloss.NewStyle().Bold(true).Render(displayID)
			}
		}
		display := displayID
		if warnTag != "" {
			tag := warnTag
			if useColor {
				tag = warnStyle.Render(warnTag)
			}
			display += tag
		}
		desc := strings.TrimSpace(item.Description)
		if desc != "" {
			if useColor {
				display += theme.Muted.Render(" - " + desc)
			} else {
				display += " - " + desc
			}
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), display))
		if len(item.Repos) == 0 {
			continue
		}
		for j, repo := range item.Repos {
			connector := "├─"
			if j == len(item.Repos)-1 {
				connector = "└─"
			}
			line := fmt.Sprintf("%s%s %s", output.Indent+output.Indent+output.Indent, connector, repo.Label)
			if useColor {
				line = theme.Muted.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
}

func WorkspaceChoiceLines(items []WorkspaceChoice, cursor int, useColor bool, theme Theme) []string {
	return collectLines(func(b *strings.Builder) {
		renderWorkspaceChoiceList(b, items, cursor, 0, useColor, theme)
	})
}

func shortWarningTag(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "dirty changes":
		return "dirty changes"
	case "unpushed commits":
		return "unpushed commits"
	case "diverged or upstream missing":
		return "diverged or upstream missing"
	case "status unknown":
		return "status unknown"
	default:
		return strings.TrimSpace(value)
	}
}

func renderBlockedChoiceList(b *strings.Builder, items []BlockedChoice, useColor bool, theme Theme) {
	for _, item := range items {
		line := fmt.Sprintf("%s%s %s", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), item.Label)
		if useColor {
			line = theme.Warn.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func (m addInputsModel) selectedWorkspace() (WorkspaceChoice, bool) {
	for _, ws := range m.workspaces {
		if ws.ID == m.workspaceID {
			return ws, true
		}
	}
	return WorkspaceChoice{}, false
}

func buildWorkspaceRepoKeys(workspaces []WorkspaceChoice) map[string]map[string]struct{} {
	out := make(map[string]map[string]struct{}, len(workspaces))
	for _, ws := range workspaces {
		keys := make(map[string]struct{})
		for _, repo := range ws.Repos {
			if strings.TrimSpace(repo.Value) == "" {
				continue
			}
			keys[repo.Value] = struct{}{}
		}
		out[ws.ID] = keys
	}
	return out
}

func filterRepoChoices(items []PromptChoice, blocked map[string]struct{}, query string) []PromptChoice {
	var out []PromptChoice
	for _, item := range items {
		if blocked != nil {
			if _, exists := blocked[item.Label]; exists {
				continue
			}
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Label), query) {
			continue
		}
		out = append(out, item)
	}
	return out
}
