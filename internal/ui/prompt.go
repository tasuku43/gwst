package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/tasuku43/gwst/internal/infra/output"
)

var ErrPromptCanceled = errors.New("prompt canceled")

type PromptChoice struct {
	Label       string
	Value       string
	Description string
	Details     []string
}

type IssueSelection struct {
	Value  string
	Branch string
	Label  string
}

type BranchSelection struct {
	Value  string
	Label  string
	Branch string
}

type createFlowStage int

const (
	createStageMode createFlowStage = iota
	createStagePreset
	createStagePresetDesc
	createStagePresetBranch
	createStageReviewRepo
	createStageReviewPRs
	createStageIssueRepo
	createStageIssueIssues
	createStageRepoSelect
	createStageRepoWorkspace
)

type WorkspaceChoice struct {
	ID            string
	WorkspacePath string
	Description   string
	Repos         []PromptChoice
	Warning       string
	WarningStrong bool
}

type BlockedChoice struct {
	Label string
}

type inputsStage int

const (
	stagePreset inputsStage = iota
	stageWorkspace
)

type inputsModel struct {
	title       string
	presets     []string
	preset      string
	workspaceID string
	label       string
	validateID  func(string) error

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
	title string

	presets []string
	tmplErr error

	modeInput textinput.Model
	filtered  []PromptChoice
	cursor    int
	err       error

	presetModel        inputsModel
	presetRepos        []string
	description        string
	descInput          textinput.Model
	branches           []string
	branchModel        branchInputModel
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

	reviewRepoModel     choiceSelectModel
	reviewPRModel       multiSelectModel
	issueRepoModel      choiceSelectModel
	issueIssueModel     issueBranchSelectModel
	repoSelectModel     choiceSelectModel
	loadReviewPRs       func(string) ([]PromptChoice, error)
	loadIssueChoices    func(string) ([]PromptChoice, error)
	loadPresetRepos     func(string) ([]string, error)
	onReposResolved     func([]string)
	validateBranch      func(string) error
	validateWorkspaceID func(string) error

	theme    Theme
	useColor bool
}

func newCreateFlowModel(title string, presets []string, tmplErr error, repoChoices []PromptChoice, repoErr error, defaultWorkspaceID string, presetName string, reviewRepos []PromptChoice, issueRepos []PromptChoice, loadReview func(string) ([]PromptChoice, error), loadIssue func(string) ([]PromptChoice, error), loadPresetRepos func(string) ([]string, error), onReposResolved func([]string), validateBranch func(string) error, validateWorkspaceID func(string) error, theme Theme, useColor bool, startMode string, selectedRepo string) createFlowModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := createFlowModel{
		stage:               createStageMode,
		presets:             presets,
		tmplErr:             tmplErr,
		repoChoices:         repoChoices,
		repoErr:             repoErr,
		defaultWorkspaceID:  defaultWorkspaceID,
		title:               title,
		modeInput:           input,
		reviewRepos:         reviewRepos,
		issueRepos:          issueRepos,
		loadReviewPRs:       loadReview,
		loadIssueChoices:    loadIssue,
		loadPresetRepos:     loadPresetRepos,
		onReposResolved:     onReposResolved,
		validateBranch:      validateBranch,
		validateWorkspaceID: validateWorkspaceID,
		theme:               theme,
		useColor:            useColor,
	}
	m.repoSelected = strings.TrimSpace(selectedRepo)
	presetName = strings.TrimSpace(presetName)
	if strings.TrimSpace(startMode) != "" {
		if startMode == "repo" && m.repoSelected != "" {
			m.mode = "repo"
			m.presetRepos = []string{m.repoSelected}
			if m.onReposResolved != nil {
				m.onReposResolved(m.presetRepos)
			}
			m.presetModel = newInputsModelWithLabel(m.title, nil, m.repoSelected, m.defaultWorkspaceID, "repo", m.validateWorkspaceID, m.theme, m.useColor)
			m.stage = createStageRepoWorkspace
		} else {
			m.startMode(startMode, presetName)
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

func (m *createFlowModel) startMode(mode, presetName string) {
	switch mode {
	case "preset":
		if m.tmplErr != nil {
			m.err = m.tmplErr
			return
		}
		if len(m.presets) == 0 {
			m.err = fmt.Errorf("no presets found")
			return
		}
		m.mode = mode
		m.stage = createStagePreset
		m.presetModel = newInputsModelWithLabel(m.title, m.presets, presetName, m.defaultWorkspaceID, "preset", m.validateWorkspaceID, m.theme, m.useColor)
	case "review":
		if len(m.reviewRepos) == 0 {
			m.err = fmt.Errorf("no GitHub repos found")
			return
		}
		m.mode = mode
		m.stage = createStageReviewRepo
		m.reviewRepoModel = newChoiceSelectModel(m.title, "repo", m.reviewRepos, m.theme, m.useColor)
	case "issue":
		if len(m.issueRepos) == 0 {
			m.err = fmt.Errorf("no repos with supported hosts found")
			return
		}
		m.mode = mode
		m.stage = createStageIssueRepo
		m.issueRepoModel = newChoiceSelectModel(m.title, "repo", m.issueRepos, m.theme, m.useColor)
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
		m.repoSelectModel = newChoiceSelectModel(m.title, "repo", m.repoChoices, m.theme, m.useColor)
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
	m.stage = createStagePresetDesc
}

func (m createFlowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = size.Height
		switch m.stage {
		case createStagePreset:
			model, _ := m.presetModel.Update(msg)
			m.presetModel = model.(inputsModel)
		case createStagePresetBranch:
			model, _ := m.branchModel.Update(msg)
			m.branchModel = model.(branchInputModel)
		case createStageRepoWorkspace:
			model, _ := m.presetModel.Update(msg)
			m.presetModel = model.(inputsModel)
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
				case "preset":
					if m.tmplErr != nil {
						m.err = m.tmplErr
						return m, tea.Quit
					}
					m.stage = createStagePreset
					m.presetModel = newInputsModel(m.title, m.presets, "", "", m.theme, m.useColor)
				case "review":
					if len(m.reviewRepos) == 0 {
						m.err = fmt.Errorf("no GitHub repos found")
						return m, tea.Quit
					}
					m.stage = createStageReviewRepo
					m.reviewRepoModel = newChoiceSelectModel(m.title, "repo", m.reviewRepos, m.theme, m.useColor)
				case "issue":
					if len(m.issueRepos) == 0 {
						m.err = fmt.Errorf("no repos with supported hosts found")
						return m, tea.Quit
					}
					m.stage = createStageIssueRepo
					m.issueRepoModel = newChoiceSelectModel(m.title, "repo", m.issueRepos, m.theme, m.useColor)
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
					m.repoSelectModel = newChoiceSelectModel(m.title, "repo", m.repoChoices, m.theme, m.useColor)
				default:
					m.err = fmt.Errorf("unknown mode: %s", m.mode)
					return m, tea.Quit
				}
				return m, nil
			}
		}
	}

	if m.stage == createStagePreset {
		model, _ := m.presetModel.Update(msg)
		m.presetModel = model.(inputsModel)
		if m.presetModel.done {
			repos, err := m.loadPresetRepos(m.presetName())
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			m.presetRepos = repos
			if m.onReposResolved != nil {
				m.onReposResolved(m.presetRepos)
			}
			m.beginDescriptionStage()
			return m, nil
		}
		return m, nil
	}

	if m.stage == createStagePresetDesc {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				m.err = ErrPromptCanceled
				return m, tea.Quit
			case tea.KeyEnter:
				m.description = strings.TrimSpace(m.descInput.Value())
				if len(m.presetRepos) == 0 {
					return m, tea.Quit
				}
				choices := make([]PromptChoice, len(m.presetRepos))
				for i, repo := range m.presetRepos {
					choices[i] = PromptChoice{Label: repo, Value: repo}
				}
				m.branchModel = newBranchInputModel(
					m.title,
					choices,
					func(index int, choice PromptChoice) string {
						label := strings.TrimSpace(choice.Label)
						if label == "" {
							return fmt.Sprintf("repo #%d", index+1)
						}
						return fmt.Sprintf("repo #%d (%s)", index+1, label)
					},
					func(PromptChoice) string { return m.workspaceID() },
					m.validateBranch,
					false,
					m.theme,
					m.useColor,
				)
				m.stage = createStagePresetBranch
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.descInput, cmd = m.descInput.Update(msg)
		return m, cmd
	}

	if m.stage == createStagePresetBranch {
		model, cmd := m.branchModel.Update(msg)
		m.branchModel = model.(branchInputModel)
		if m.branchModel.err != nil {
			m.err = m.branchModel.err
			return m, tea.Quit
		}
		if m.branchModel.done {
			selections := m.branchModel.Selections()
			m.branches = make([]string, len(selections))
			for i, selection := range selections {
				m.branches[i] = selection.Branch
			}
			return m, tea.Quit
		}
		return m, cmd
	}

	if m.stage == createStageReviewRepo {
		model, _ := m.reviewRepoModel.Update(msg)
		m.reviewRepoModel = model.(choiceSelectModel)
		if m.reviewRepoModel.done {
			m.reviewRepo = m.reviewRepoModel.value
			if m.onReposResolved != nil {
				m.onReposResolved([]string{m.reviewRepo})
			}
			choices, err := m.loadReviewPRs(m.reviewRepo)
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			if len(choices) == 0 {
				m.err = fmt.Errorf("no pull requests found")
				return m, tea.Quit
			}
			m.reviewPRModel = newMultiSelectModel(m.title, "pull request", choices, m.theme, m.useColor)
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
			if m.onReposResolved != nil {
				m.onReposResolved([]string{m.issueRepo})
			}
			choices, err := m.loadIssueChoices(m.issueRepo)
			if err != nil {
				m.err = err
				return m, tea.Quit
			}
			if len(choices) == 0 {
				m.err = fmt.Errorf("no issues found")
				return m, tea.Quit
			}
			m.issueIssueModel = newIssueBranchSelectModel(m.title, "issue", choices, m.validateBranch, m.theme, m.useColor)
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
			m.presetRepos = []string{m.repoSelected}
			if m.onReposResolved != nil {
				m.onReposResolved(m.presetRepos)
			}
			m.presetModel = newInputsModelWithLabel(m.title, nil, m.repoSelected, m.defaultWorkspaceID, "repo", m.validateWorkspaceID, m.theme, m.useColor)
			m.stage = createStageRepoWorkspace
		}
		return m, nil
	}

	if m.stage == createStageRepoWorkspace {
		model, _ := m.presetModel.Update(msg)
		m.presetModel = model.(inputsModel)
		if m.presetModel.done {
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
	if m.stage == createStagePreset {
		return m.presetModel.View()
	}
	if m.stage == createStagePresetDesc {
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
	if m.stage == createStagePresetBranch {
		labelSelection := promptLabel(m.theme, m.useColor, m.selectionLabel())
		labelWorkspace := promptLabel(m.theme, m.useColor, "workspace id")
		labelDesc := promptLabel(m.theme, m.useColor, "description")
		lines := []string{
			fmt.Sprintf("%s: %s", labelSelection, m.selectionValue()),
			fmt.Sprintf("%s: %s", labelWorkspace, m.workspaceID()),
		}
		if m.description != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", labelDesc, m.description))
		}
		return m.branchModel.ViewWithHeader(lines...)
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
			return m.issueIssueModel.branchModel.ViewWithHeader(fmt.Sprintf("%s: %s", labelRepo, m.issueRepo))
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
		return m.presetModel.View()
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
		{Label: "repo", Value: "repo", Description: "1 repo only"},
		{Label: "issue", Value: "issue", Description: "From an issue (multi-select, GitHub only)"},
		{Label: "review", Value: "review", Description: "From a review request (multi-select, GitHub only)"},
		{Label: "preset", Value: "preset", Description: "From preset"},
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

func (m createFlowModel) presetName() string {
	return strings.TrimSpace(m.presetModel.preset)
}

func (m createFlowModel) workspaceID() string {
	return m.presetModel.currentWorkspaceID()
}

func (m createFlowModel) selectionLabel() string {
	if m.mode == "repo" {
		return "repo"
	}
	return "preset"
}

func (m createFlowModel) selectionValue() string {
	if m.mode == "repo" {
		return m.repoSelected
	}
	return m.presetName()
}

func newInputsModel(title string, presets []string, presetName string, workspaceID string, theme Theme, useColor bool) inputsModel {
	return newInputsModelWithLabel(title, presets, presetName, workspaceID, "preset", nil, theme, useColor)
}

func newInputsModelWithLabel(title string, presets []string, presetName string, workspaceID string, label string, validateID func(string) error, theme Theme, useColor bool) inputsModel {
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

	stage := stagePreset
	if strings.TrimSpace(presetName) != "" {
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
		presets:     presets,
		preset:      presetName,
		workspaceID: workspaceID,
		label:       label,
		validateID:  validateID,
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
			if m.stage == stagePreset {
				if len(m.filtered) == 0 {
					return m, nil
				}
				m.preset = m.filtered[m.cursor]
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
				if m.validateID != nil {
					if err := m.validateID(value); err != nil {
						m.errorLine = err.Error()
						return m, nil
					}
				}
				m.workspaceID = value
				m.done = true
				return m, tea.Quit
			}
		case tea.KeyUp:
			if m.stage == stagePreset && m.cursor > 0 {
				m.cursor--
				return m, nil
			}
		case tea.KeyDown:
			if m.stage == stagePreset && m.cursor < len(m.filtered)-1 {
				m.cursor++
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	if m.stage == stagePreset {
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
		label = "preset"
	}
	labelPreset := promptLabel(m.theme, m.useColor, label)
	var promptLines []string
	if m.stage == stagePreset {
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", labelPreset, m.search.View()))
	} else {
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", labelPreset, m.preset))
	}

	if m.stage == stageWorkspace {
		label := promptLabel(m.theme, m.useColor, "workspace id")
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", label, m.idInput.View()))
	} else if strings.TrimSpace(m.workspaceID) != "" {
		label := promptLabel(m.theme, m.useColor, "workspace id")
		promptLines = append(promptLines, fmt.Sprintf("%s: %s", label, m.workspaceID))
	}
	frame.SetInputsPrompt(promptLines...)

	if m.stage == stagePreset {
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

func formatInputsHeader(title, presetName, workspaceID string) string {
	var parts []string
	if strings.TrimSpace(presetName) != "" {
		parts = append(parts, fmt.Sprintf("preset: %s", presetName))
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
		return append([]string(nil), m.presets...)
	}
	var out []string
	for _, item := range m.presets {
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
	rawAfter     bool
	input        textinput.Model
	value        bool
	err          error
	done         bool
}

// confirmInlineLineModel renders a single inline prompt line (no Frame/section headers).
// Intended for embedding prompts inside an existing section (e.g. Plan).
type confirmInlineLineModel struct {
	label    string
	theme    Theme
	useColor bool
	input    textinput.Model
	value    bool
	err      error
	done     bool
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
		rawAfter:     false,
		input:        ti,
	}
}

func newConfirmInlineLineModel(label string, theme Theme, useColor bool) confirmInlineLineModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "y/n"
	ti.Focus()
	if useColor {
		ti.PlaceholderStyle = theme.Muted
	}
	return confirmInlineLineModel{
		label:    label,
		theme:    theme,
		useColor: useColor,
		input:    ti,
	}
}

func newConfirmInlineModelWithRawAfterPrompt(label string, theme Theme, useColor bool, inputsRaw []string) confirmInlineModel {
	model := newConfirmInlineModel(label, theme, useColor, false, nil, inputsRaw)
	model.rawAfter = true
	return model
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
	} else if m.rawAfter {
		frame.SetInputsPrompt(line)
		if len(m.inputsRaw) > 0 {
			frame.AppendInputsRaw(m.inputsRaw...)
		}
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

func (m confirmInlineLineModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m confirmInlineLineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m confirmInlineLineModel) View() string {
	label := promptLabel(m.theme, m.useColor, m.label)
	line := fmt.Sprintf("%s (y/n)", label)

	prefix := output.StepPrefix + " "
	if m.useColor {
		prefix = m.theme.Accent.Render(output.StepPrefix) + " "
	}
	connector := mutedToken(m.theme, m.useColor, output.LogConnector)
	return output.Indent + prefix + line + "\n" +
		fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, connector, m.input.View())
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

type presetRepoSelectModel struct {
	title      string
	presetName string
	choices    []PromptChoice
	filtered   []PromptChoice
	selected   []string
	addedNote  string

	theme    Theme
	useColor bool

	nameInput textinput.Model
	repoInput textinput.Model

	stage     presetRepoStage
	cursor    int
	err       error
	errorLine string

	height int
}

type presetRepoStage int

const (
	stagePresetName presetRepoStage = iota
	stageRepoSelect
)

func newPresetRepoSelectModel(title string, presetName string, choices []PromptChoice, theme Theme, useColor bool) presetRepoSelectModel {
	repoInput := textinput.New()
	repoInput.Prompt = ""
	repoInput.Placeholder = "search"
	if useColor {
		repoInput.PlaceholderStyle = theme.Muted
	}

	nameInput := textinput.New()
	nameInput.Prompt = ""
	nameInput.Placeholder = "preset name"
	nameInput.SetValue(presetName)
	if useColor {
		nameInput.PlaceholderStyle = theme.Muted
	}

	stage := stageRepoSelect
	if strings.TrimSpace(presetName) == "" {
		stage = stagePresetName
		nameInput.Focus()
	} else {
		repoInput.Focus()
	}

	m := presetRepoSelectModel{
		title:      title,
		presetName: presetName,
		choices:    choices,
		theme:      theme,
		useColor:   useColor,
		nameInput:  nameInput,
		repoInput:  repoInput,
		stage:      stage,
	}
	m.filtered = m.filterChoices()
	return m
}

func (m presetRepoSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m presetRepoSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.stage == stagePresetName {
				if strings.TrimSpace(m.nameInput.Value()) == "" {
					m.errorLine = "preset name is required"
					return m, nil
				}
				m.presetName = strings.TrimSpace(m.nameInput.Value())
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
			if m.stage == stagePresetName {
				value := strings.TrimSpace(m.nameInput.Value())
				if value == "" {
					m.errorLine = "preset name is required"
					return m, nil
				}
				m.presetName = value
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
	if m.stage == stagePresetName {
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

func (m presetRepoSelectModel) View() string {
	frame := NewFrame(m.theme, m.useColor)

	labelName := promptLabel(m.theme, m.useColor, "preset name")
	presetName := m.presetName
	if m.stage == stagePresetName {
		presetName = m.nameInput.View()
	}
	labelRepo := promptLabel(m.theme, m.useColor, "repo")
	repoInput := m.repoInput.View()
	if m.stage == stagePresetName {
		repoInput = ""
	}
	frame.SetInputsPrompt(
		fmt.Sprintf("%s: %s", labelName, presetName),
		fmt.Sprintf("%s: %s", labelRepo, repoInput),
	)

	selectedLines := collectLines(func(b *strings.Builder) {
		renderSelectedRepoTree(b, m.selected, m.useColor, m.theme)
	})
	infoLines := 1 + len(selectedLines) + 1
	if m.errorLine != "" {
		infoLines++
	}
	// +1 for the inline "finish" help line appended under Inputs.
	maxLines := listMaxLines(m.height, 3, infoLines)
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
	frame.AppendInputsRaw(
		fmt.Sprintf("%s%s finish: Ctrl+D or type \"done\"", output.Indent, infoPrefix),
	)
	return frame.Render()
}

func (m presetRepoSelectModel) filterChoices() []PromptChoice {
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
	// +1 for the inline "finish" help line appended under Inputs.
	maxLines := listMaxLines(height, len(lines)+1, infoLines)
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
	frame.AppendInputsRaw(
		fmt.Sprintf("%s%s finish: Ctrl+D or type \"done\"", output.Indent, infoPrefix),
	)
	return frame.Render()
}

type branchInputModel struct {
	title             string
	items             []PromptChoice
	index             int
	input             textinput.Model
	errorLine         string
	err               error
	done              bool
	selections        []BranchSelection
	usedBranches      map[string]int
	itemLabel         func(int, PromptChoice) string
	defaultBranch     func(PromptChoice) string
	validateBranch    func(string) error
	ensureUnique      bool
	separateInputLine bool
	theme             Theme
	useColor          bool
}

func newBranchInputModel(title string, items []PromptChoice, itemLabel func(int, PromptChoice) string, defaultBranch func(PromptChoice) string, validateBranch func(string) error, ensureUnique bool, theme Theme, useColor bool) branchInputModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "branch"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := branchInputModel{
		title:          title,
		items:          items,
		input:          input,
		itemLabel:      itemLabel,
		defaultBranch:  defaultBranch,
		validateBranch: validateBranch,
		ensureUnique:   ensureUnique,
		theme:          theme,
		useColor:       useColor,
		selections:     make([]BranchSelection, len(items)),
	}
	if ensureUnique {
		m.usedBranches = map[string]int{}
	}
	if len(items) > 0 && m.defaultBranch != nil {
		value := strings.TrimSpace(m.defaultBranch(items[0]))
		if value != "" {
			m.input.SetValue(value)
			m.input.CursorEnd()
		}
	}
	return m
}

func (m branchInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m branchInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = ErrPromptCanceled
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.items) == 0 {
				m.done = true
				return m, tea.Quit
			}
			value := strings.TrimSpace(m.input.Value())
			if value == "" && m.defaultBranch != nil {
				value = strings.TrimSpace(m.defaultBranch(m.items[m.index]))
			}
			if m.validateBranch != nil {
				if err := m.validateBranch(value); err != nil {
					m.errorLine = err.Error()
					return m, nil
				}
			}
			if m.ensureUnique {
				if prev, exists := m.usedBranches[value]; exists {
					m.errorLine = fmt.Sprintf("branch %q already used for %s; re-enter", value, m.items[prev].Label)
					return m, nil
				}
			}
			m.selections[m.index] = BranchSelection{
				Value:  m.items[m.index].Value,
				Label:  m.items[m.index].Label,
				Branch: value,
			}
			if m.ensureUnique {
				m.usedBranches[value] = m.index
			}
			m.errorLine = ""
			m.index++
			if m.index >= len(m.items) {
				m.done = true
				return m, tea.Quit
			}
			if m.defaultBranch != nil {
				m.input.SetValue(m.defaultBranch(m.items[m.index]))
				m.input.CursorEnd()
			} else {
				m.input.SetValue("")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if strings.TrimSpace(m.input.Value()) != "" {
		m.errorLine = ""
	}
	return m, cmd
}

func (m branchInputModel) ViewWithHeader(headerLines ...string) string {
	frame := NewFrame(m.theme, m.useColor)
	lines := append([]string(nil), headerLines...)
	if len(m.items) == 0 {
		frame.SetInputsPrompt(lines...)
		return frame.Render()
	}

	if m.separateInputLine {
		frame.SetInputsPrompt(lines...)
		maxIndex := m.index
		if maxIndex > len(m.items)-1 {
			maxIndex = len(m.items) - 1
		}
		for i := 0; i <= maxIndex; i++ {
			item := m.items[i]
			label := "item"
			if m.itemLabel != nil {
				label = m.itemLabel(i, item)
			} else if strings.TrimSpace(item.Label) != "" {
				label = item.Label
			}
			frame.AppendInputsPrompt(label)

			value := ""
			if i < m.index {
				value = m.selections[i].Branch
			} else {
				value = m.input.View()
			}
			connector := mutedToken(m.theme, m.useColor, output.LogConnector)
			frame.AppendInputsRaw(fmt.Sprintf("%s%s branch: %s", output.Indent+output.Indent, connector, value))
		}
		if m.errorLine != "" {
			msg := m.errorLine
			if m.useColor {
				msg = m.theme.Error.Render(msg)
			}
			frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
		}
		return frame.Render()
	}

	maxIndex := m.index
	if maxIndex > len(m.items)-1 {
		maxIndex = len(m.items) - 1
	}
	for i := 0; i <= maxIndex; i++ {
		item := m.items[i]
		label := "item"
		if m.itemLabel != nil {
			label = m.itemLabel(i, item)
		} else if strings.TrimSpace(item.Label) != "" {
			label = item.Label
		}
		labelBranch := promptLabel(m.theme, m.useColor, fmt.Sprintf("branch for %s", label))
		if i < m.index {
			lines = append(lines, fmt.Sprintf("%s: %s", labelBranch, m.selections[i].Branch))
		} else {
			lines = append(lines, fmt.Sprintf("%s: %s", labelBranch, m.input.View()))
		}
	}
	frame.SetInputsPrompt(lines...)
	if m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		frame.AppendInfoRaw(fmt.Sprintf("%s%s %s", output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}
	return frame.Render()
}

func (m branchInputModel) View() string {
	return m.ViewWithHeader()
}

func (m branchInputModel) Selections() []BranchSelection {
	return append([]BranchSelection(nil), m.selections...)
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
	selectedIssues []IssueSelection
	branchModel    branchInputModel

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
		case tea.KeyDown:
			if m.stage == issueBranchStageSelect && m.cursor < len(m.filtered)-1 {
				m.cursor++
				return m, nil
			}
		case tea.KeyCtrlD:
			if m.stage == issueBranchStageSelect {
				if len(m.selected) == 0 {
					m.errorLine = fmt.Sprintf("select at least one %s", m.label)
					return m, nil
				}
				m = m.startBranchInput()
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
					m = m.startBranchInput()
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

	if m.stage == issueBranchStageEdit {
		model, cmd := m.branchModel.Update(msg)
		m.branchModel = model.(branchInputModel)
		if m.branchModel.err != nil {
			m.err = m.branchModel.err
			return m, tea.Quit
		}
		if m.branchModel.done {
			selections := m.branchModel.Selections()
			m.selectedIssues = make([]IssueSelection, len(selections))
			for i, selection := range selections {
				m.selectedIssues[i] = IssueSelection{
					Value:  selection.Value,
					Branch: selection.Branch,
					Label:  selection.Label,
				}
			}
			m.done = true
			return m, tea.Quit
		}
		return m, cmd
	}

	return m, nil
}

func (m issueBranchSelectModel) View() string {
	if m.stage == issueBranchStageEdit {
		return m.branchModel.ViewWithHeader()
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

func (m issueBranchSelectModel) startBranchInput() issueBranchSelectModel {
	m.stage = issueBranchStageEdit
	m.branchModel = newBranchInputModel(
		m.title,
		m.selected,
		func(index int, choice PromptChoice) string {
			label := strings.TrimSpace(choice.Label)
			if label == "" {
				return fmt.Sprintf("issue #%d", index+1)
			}
			return fmt.Sprintf("issue #%d (%s)", index+1, label)
		},
		func(choice PromptChoice) string {
			return defaultIssueBranch(choice.Value)
		},
		m.validateBranch,
		true,
		m.theme,
		m.useColor,
	)
	m.branchModel.separateInputLine = true
	m.errorLine = ""
	return m
}

func defaultIssueBranch(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "issue/"
	}
	return fmt.Sprintf("issue/%s", value)
}

type presetNameModel struct {
	title     string
	theme     Theme
	useColor  bool
	input     textinput.Model
	value     string
	err       error
	errorLine string
}

func newPresetNameModel(title string, defaultValue string, theme Theme, useColor bool) presetNameModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "preset name"
	input.SetValue(defaultValue)
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	return presetNameModel{
		title:    title,
		theme:    theme,
		useColor: useColor,
		input:    input,
	}
}

func (m presetNameModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m presetNameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m presetNameModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "preset name")
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

type windowSizeModel struct {
	inner tea.Model
}

func (m windowSizeModel) Init() tea.Cmd {
	return m.inner.Init()
}

func (m windowSizeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		setWrapWidth(size.Width)
	}
	next, cmd := m.inner.Update(msg)
	m.inner = next
	return m, cmd
}

func (m windowSizeModel) View() string {
	return m.inner.View()
}

func unwrapWindowSizeModel(model tea.Model) tea.Model {
	if wrapped, ok := model.(windowSizeModel); ok {
		return wrapped.inner
	}
	return model
}

func runProgram(model tea.Model) (tea.Model, error) {
	setStableLayout(true)
	defer setStableLayout(false)
	out, err := tea.NewProgram(windowSizeModel{inner: model}).Run()
	if err != nil {
		return out, err
	}
	return unwrapWindowSizeModel(out), nil
}

func runProgramWithOutput(model tea.Model, out io.Writer) (tea.Model, error) {
	setStableLayout(true)
	defer setStableLayout(false)
	final, err := tea.NewProgram(windowSizeModel{inner: model}, tea.WithOutput(out)).Run()
	if err != nil {
		return final, err
	}
	return unwrapWindowSizeModel(final), nil
}

func runProgramWithIO(model tea.Model, in io.Reader, out io.Writer, altScreen bool) (tea.Model, error) {
	setStableLayout(true)
	defer setStableLayout(false)
	opts := []tea.ProgramOption{tea.WithInput(in), tea.WithOutput(out)}
	if altScreen {
		opts = append(opts, tea.WithAltScreen())
	}
	final, err := tea.NewProgram(windowSizeModel{inner: model}, opts...).Run()
	if err != nil {
		return final, err
	}
	return unwrapWindowSizeModel(final), nil
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

func wrapRawLineToWidth(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	prefix, rest, ok := splitRawLinePrefix(line)
	if !ok {
		prefix = ""
		rest = line
	}
	prefixWidth := lipgloss.Width(prefix)
	available := width - prefixWidth
	if available <= 0 {
		return []string{line}
	}
	tail := "..."
	if available < ansi.StringWidth(tail) {
		tail = ""
	}
	single := strings.ReplaceAll(rest, "\n", " ")
	truncated := ansi.Truncate(single, available, tail)
	return []string{prefix + truncated}
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

type workspaceRepoSelection struct {
	WorkspaceIndex int
	RepoIndex      int
	Path           string
}

type workspaceRepoSelectModel struct {
	title      string
	workspaces []WorkspaceChoice
	theme      Theme
	useColor   bool

	input      textinput.Model
	filtered   []WorkspaceChoice
	selections []workspaceRepoSelection
	cursor     int
	err        error

	selectedPath string
	height       int
	lastQuery    string
}

func newWorkspaceRepoSelectModel(title string, workspaces []WorkspaceChoice, theme Theme, useColor bool) workspaceRepoSelectModel {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "search"
	input.Focus()
	if useColor {
		input.PlaceholderStyle = theme.Muted
	}
	m := workspaceRepoSelectModel{
		title:      title,
		workspaces: workspaces,
		theme:      theme,
		useColor:   useColor,
		input:      input,
	}
	m.filtered = m.filterWorkspaces()
	m.rebuildSelections()
	m.lastQuery = strings.TrimSpace(m.input.Value())
	if m.lastQuery != "" && len(m.selections) > 0 {
		m.cursor = m.bestSelectionIndex(m.lastQuery)
	}
	return m
}

func (m workspaceRepoSelectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m workspaceRepoSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.selections)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if len(m.selections) == 0 {
				return m, nil
			}
			m.selectedPath = m.selections[m.cursor].Path
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filtered = m.filterWorkspaces()
	m.rebuildSelections()
	nextQuery := strings.TrimSpace(m.input.Value())
	if nextQuery != m.lastQuery {
		if nextQuery != "" && len(m.selections) > 0 {
			m.cursor = m.bestSelectionIndex(nextQuery)
		}
		m.lastQuery = nextQuery
	}
	if m.cursor >= len(m.selections) {
		m.cursor = max(0, len(m.selections)-1)
	}
	return m, cmd
}

func (m workspaceRepoSelectModel) View() string {
	frame := NewFrame(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "workspace")
	frame.SetInputsPrompt(fmt.Sprintf("%s: %s", label, m.input.View()))
	maxLines := listMaxLines(m.height, 1, 0)
	rawLines := collectLines(func(b *strings.Builder) {
		renderWorkspaceRepoChoiceList(b, m.filtered, m.cursor, maxLines, m.useColor, m.theme)
	})
	frame.AppendInputsRaw(rawLines...)
	return frame.Render()
}

func (m *workspaceRepoSelectModel) rebuildSelections() {
	m.selections = m.selections[:0]
	for wsIndex, ws := range m.filtered {
		path := strings.TrimSpace(ws.WorkspacePath)
		if path == "" {
			path = strings.TrimSpace(ws.ID)
		}
		if path != "" {
			m.selections = append(m.selections, workspaceRepoSelection{
				WorkspaceIndex: wsIndex,
				RepoIndex:      -1,
				Path:           path,
			})
		}
		for repoIndex, repo := range ws.Repos {
			repoPath := strings.TrimSpace(repo.Value)
			if repoPath == "" {
				continue
			}
			m.selections = append(m.selections, workspaceRepoSelection{
				WorkspaceIndex: wsIndex,
				RepoIndex:      repoIndex,
				Path:           repoPath,
			})
		}
	}
}

func (m workspaceRepoSelectModel) filterWorkspaces() []WorkspaceChoice {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return cloneWorkspaceChoices(m.workspaces)
	}
	var out []WorkspaceChoice
	for _, item := range m.workspaces {
		wsMatch := workspaceChoiceMatches(item, q)
		var repos []PromptChoice
		if wsMatch {
			repos = append(repos, item.Repos...)
		} else {
			for _, repo := range item.Repos {
				if repoChoiceMatches(repo, q) {
					repos = append(repos, repo)
				}
			}
		}
		if wsMatch || len(repos) > 0 {
			cloned := item
			cloned.Repos = repos
			out = append(out, cloned)
		}
	}
	return out
}

func (m workspaceRepoSelectModel) bestSelectionIndex(q string) int {
	if len(m.selections) == 0 {
		return 0
	}
	bestIndex := 0
	bestScore := -1
	query := strings.ToLower(strings.TrimSpace(q))
	for i, sel := range m.selections {
		text := m.selectionSearchText(sel)
		score := fuzzyScore(strings.ToLower(text), query)
		if score < 0 {
			continue
		}
		if bestScore == -1 || score < bestScore || (score == bestScore && sel.RepoIndex >= 0 && m.selections[bestIndex].RepoIndex < 0) {
			bestScore = score
			bestIndex = i
		}
	}
	if bestScore == -1 {
		return 0
	}
	return bestIndex
}

func (m workspaceRepoSelectModel) selectionSearchText(sel workspaceRepoSelection) string {
	if sel.WorkspaceIndex < 0 || sel.WorkspaceIndex >= len(m.filtered) {
		return ""
	}
	ws := m.filtered[sel.WorkspaceIndex]
	if sel.RepoIndex < 0 {
		return workspaceSearchText(ws)
	}
	if sel.RepoIndex >= len(ws.Repos) {
		return workspaceSearchText(ws)
	}
	repo := ws.Repos[sel.RepoIndex]
	return repoSearchText(ws, repo)
}

func workspaceSearchText(ws WorkspaceChoice) string {
	parts := []string{ws.ID, ws.Description}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func repoSearchText(ws WorkspaceChoice, repo PromptChoice) string {
	parts := []string{
		repo.Label,
		repo.Description,
	}
	if len(repo.Details) > 0 {
		parts = append(parts, repo.Details...)
	}
	if repo.Value != "" {
		parts = append(parts, repo.Value)
	}
	parts = append(parts, ws.ID, ws.Description)
	return strings.TrimSpace(strings.Join(parts, " "))
}

func cloneWorkspaceChoices(items []WorkspaceChoice) []WorkspaceChoice {
	out := make([]WorkspaceChoice, 0, len(items))
	for _, item := range items {
		cloned := item
		if len(item.Repos) > 0 {
			cloned.Repos = append([]PromptChoice(nil), item.Repos...)
		}
		out = append(out, cloned)
	}
	return out
}

func workspaceChoiceMatches(item WorkspaceChoice, q string) bool {
	if fuzzyMatch(strings.ToLower(item.ID), q) {
		return true
	}
	if fuzzyMatch(strings.ToLower(item.Description), q) {
		return true
	}
	return false
}

func repoChoiceMatches(repo PromptChoice, q string) bool {
	if fuzzyMatch(strings.ToLower(repo.Label), q) {
		return true
	}
	if fuzzyMatch(strings.ToLower(repo.Value), q) {
		return true
	}
	if fuzzyMatch(strings.ToLower(repo.Description), q) {
		return true
	}
	for _, detail := range repo.Details {
		if fuzzyMatch(strings.ToLower(detail), q) {
			return true
		}
	}
	return false
}

func fuzzyMatch(haystack, needle string) bool {
	q := strings.ToLower(strings.TrimSpace(needle))
	if q == "" {
		return true
	}
	q = strings.Join(strings.Fields(q), "")
	if q == "" {
		return true
	}
	if haystack == "" {
		return false
	}
	j := 0
	for i := 0; i < len(haystack) && j < len(q); i++ {
		if haystack[i] == q[j] {
			j++
		}
	}
	return j == len(q)
}

func fuzzyScore(haystack, needle string) int {
	q := strings.ToLower(strings.TrimSpace(needle))
	if q == "" {
		return 0
	}
	q = strings.Join(strings.Fields(q), "")
	if q == "" {
		return 0
	}
	if haystack == "" {
		return -1
	}
	j := 0
	first := -1
	last := -1
	prev := -1
	gaps := 0
	for i := 0; i < len(haystack) && j < len(q); i++ {
		if haystack[i] == q[j] {
			if first == -1 {
				first = i
			}
			if prev != -1 {
				gaps += i - prev - 1
			}
			prev = i
			last = i
			j++
		}
	}
	if j != len(q) {
		return -1
	}
	span := 0
	if first >= 0 && last >= 0 {
		span = last - first
	}
	return gaps + span + first*2
}

type workspaceMultiSelectModel struct {
	title       string
	workspaces  []WorkspaceChoice
	blocked     []BlockedChoice
	filtered    []WorkspaceChoice
	selected    []WorkspaceChoice
	selectedIDs []string
	cursor      int
	err         error
	errorLine   string
	canceled    bool
	finalizing  bool

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
			m.finalizing = true
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
					m.errorLine = "select at least one workspace"
					return m, nil
				}
				m.finalizing = true
				return m, tea.Quit
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
	if m.finalizing {
		frame := NewFrame(m.theme, m.useColor)
		frame.NoBlankAfterInfo = true
		label := promptLabel(m.theme, m.useColor, "workspace")
		frame.SetInputsPrompt(fmt.Sprintf("%s: %s", label, m.input.View()))
		selectedLines := collectLines(func(b *strings.Builder) {
			renderSelectedWorkspaceTree(b, m.selected, m.useColor, m.theme)
		})
		if m.useColor {
			frame.SetInfo(m.theme.Accent.Render("selected"))
		} else {
			frame.SetInfo("selected")
		}
		frame.AppendInfoRaw(selectedLines...)
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
	// +1 for the inline "finish" help line appended under Inputs.
	maxLines := listMaxLines(m.height, 2, infoLines)
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
	frame.AppendInputsRaw(fmt.Sprintf("%s%s finish: Ctrl+D or type \"done\"", output.Indent, infoPrefix))

	if len(blockedLines) > 0 {
		frame.AppendInfo("blocked workspaces")
		frame.AppendInfoRaw(blockedLines...)
	}
	return frame.Render()
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

	width := currentWrapWidth()
	groups := make([]workspaceRepoChoiceGroup, 0, len(items))
	for i := range items {
		item := items[i]
		display := item
		if i == cursor && useColor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		line := fmt.Sprintf("%s%s %s", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), display)
		groups = append(groups, workspaceRepoChoiceGroup{lines: wrapRawLineToWidth(line, width)})
	}

	if cursor < 0 || cursor >= len(groups) {
		cursor = 0
	}
	if maxVisible <= 0 {
		maxVisible = 1
	}
	if len(groups[cursor].lines) > maxVisible {
		start, end := listWindow(len(groups[cursor].lines), 0, maxVisible)
		for i := start; i < end; i++ {
			b.WriteString(groups[cursor].lines[i])
			b.WriteString("\n")
		}
		return
	}
	startGroup, endGroup := groupWindowByLineBudget(groups, cursor, maxVisible)
	for gi := startGroup; gi < endGroup; gi++ {
		for _, line := range groups[gi].lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
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

	width := currentWrapWidth()
	groups := make([]workspaceRepoChoiceGroup, 0, len(items))
	for i := range items {
		item := items[i]
		display := item.Label
		if i == cursor && useColor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		desc := strings.TrimSpace(item.Description)
		if desc != "" {
			if useColor {
				display += theme.Muted.Render(" - " + desc)
			} else {
				display += " - " + desc
			}
		}
		line := fmt.Sprintf("%s%s %s", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), display)
		groups = append(groups, workspaceRepoChoiceGroup{lines: wrapRawLineToWidth(line, width)})
	}

	if cursor < 0 || cursor >= len(groups) {
		cursor = 0
	}
	if maxVisible <= 0 {
		maxVisible = 1
	}
	if len(groups[cursor].lines) > maxVisible {
		start, end := listWindow(len(groups[cursor].lines), 0, maxVisible)
		for i := start; i < end; i++ {
			b.WriteString(groups[cursor].lines[i])
			b.WriteString("\n")
		}
		return
	}
	startGroup, endGroup := groupWindowByLineBudget(groups, cursor, maxVisible)
	for gi := startGroup; gi < endGroup; gi++ {
		for _, line := range groups[gi].lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
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

type workspaceRepoChoiceGroup struct {
	lines []string
}

func buildWorkspaceRepoChoiceGroups(items []WorkspaceChoice, cursor int, useColor bool, theme Theme) ([]workspaceRepoChoiceGroup, int, int) {
	var groups []workspaceRepoChoiceGroup
	width := currentWrapWidth()

	baseIndent := output.Indent + output.Indent
	selectIndent := func(selected bool) string {
		if !selected || useColor {
			return baseIndent
		}
		return ">" + baseIndent[1:]
	}

	cursorWorkspace := -1
	cursorLine := -1

	selectIndex := 0
	for wsIndex, item := range items {
		var g workspaceRepoChoiceGroup

		workspaceConnector := "├─"
		isLastWorkspace := wsIndex == len(items)-1
		if isLastWorkspace {
			workspaceConnector = "└─"
		}
		connectorToken := workspaceConnector
		if useColor {
			connectorToken = theme.Muted.Render(workspaceConnector)
		}
		workspaceStem := "│ "
		if isLastWorkspace {
			workspaceStem = "  "
		}

		selectedWorkspace := selectIndex == cursor
		displayID := item.ID
		warnValue := shortWarningTag(item.Warning)
		hasWarn := strings.TrimSpace(warnValue) != "" && strings.TrimSpace(strings.ToLower(warnValue)) != "clean"
		warnStyle := theme.Warn
		if item.WarningStrong {
			warnStyle = theme.Error
		}
		warnTag := ""
		if hasWarn {
			warnTag = "[" + warnValue + "]"
		}
		if useColor && selectedWorkspace {
			displayID = lipgloss.NewStyle().Bold(true).Render(displayID)
		}
		display := displayID
		if warnTag != "" {
			tag := warnTag
			if useColor {
				if hasWarn {
					tag = warnStyle.Render(warnTag)
				} else {
					tag = theme.Accent.Render(warnTag)
				}
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
		indent := selectIndent(selectedWorkspace)
		workspaceLine := fmt.Sprintf("%s%s %s", indent, connectorToken, display)
		if selectedWorkspace {
			cursorWorkspace = wsIndex
			cursorLine = len(g.lines)
		}
		g.lines = append(g.lines, wrapRawLineToWidth(workspaceLine, width)...)
		selectIndex++

		for repoIndex, repo := range item.Repos {
			repoConnector := "├─"
			isLastRepo := repoIndex == len(item.Repos)-1
			if isLastRepo {
				repoConnector = "└─"
			}
			selectedRepo := selectIndex == cursor
			stemToken := workspaceStem
			connector := repoConnector
			if useColor {
				stemToken = theme.Muted.Render(workspaceStem)
				connector = theme.Muted.Render(repoConnector)
			}
			repoLabel := repo.Label
			if useColor && selectedRepo {
				repoLabel = lipgloss.NewStyle().Bold(true).Render(repoLabel)
			}
			indent := selectIndent(selectedRepo)
			repoLine := fmt.Sprintf("%s%s%s %s", indent, stemToken, connector, repoLabel)
			if selectedRepo {
				cursorWorkspace = wsIndex
				cursorLine = len(g.lines)
			}
			g.lines = append(g.lines, wrapRawLineToWidth(repoLine, width)...)
			selectIndex++

			if len(repo.Details) == 0 {
				continue
			}
			repoStem := "│  "
			if isLastRepo {
				repoStem = "   "
			}
			for _, detail := range repo.Details {
				if strings.TrimSpace(detail) == "" {
					continue
				}
				detailLine := fmt.Sprintf("%s%s%s%s", output.Indent+output.Indent, workspaceStem, repoStem, detail)
				if useColor {
					detailLine = theme.Muted.Render(detailLine)
				}
				g.lines = append(g.lines, wrapRawLineToWidth(detailLine, width)...)
			}
		}
		groups = append(groups, g)
	}
	return groups, cursorWorkspace, cursorLine
}

func groupWindowByLineBudget(groups []workspaceRepoChoiceGroup, cursorWorkspace int, maxLines int) (int, int) {
	if len(groups) == 0 {
		return 0, 0
	}
	if cursorWorkspace < 0 || cursorWorkspace >= len(groups) {
		cursorWorkspace = 0
	}
	start := cursorWorkspace
	end := cursorWorkspace + 1
	budget := len(groups[cursorWorkspace].lines)

	for {
		canAddAbove := start > 0 && budget+len(groups[start-1].lines) <= maxLines
		canAddBelow := end < len(groups) && budget+len(groups[end].lines) <= maxLines
		if !canAddAbove && !canAddBelow {
			break
		}
		if canAddAbove && canAddBelow {
			aboveSpan := cursorWorkspace - start
			belowSpan := (end - 1) - cursorWorkspace
			if aboveSpan <= belowSpan {
				start--
				budget += len(groups[start].lines)
				continue
			}
			budget += len(groups[end].lines)
			end++
			continue
		}
		if canAddAbove {
			start--
			budget += len(groups[start].lines)
			continue
		}
		budget += len(groups[end].lines)
		end++
	}
	return start, end
}

func renderWorkspaceRepoChoiceList(b *strings.Builder, items []WorkspaceChoice, cursor int, maxVisible int, useColor bool, theme Theme) {
	groups, cursorWorkspace, cursorLine := buildWorkspaceRepoChoiceGroups(items, cursor, useColor, theme)
	if len(groups) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}

	if maxVisible <= 0 {
		maxVisible = 1
	}

	if cursorWorkspace < 0 || cursorWorkspace >= len(groups) {
		cursorWorkspace = 0
		cursorLine = 0
	}
	cursorGroup := groups[cursorWorkspace]
	if cursorLine < 0 || cursorLine >= len(cursorGroup.lines) {
		cursorLine = 0
	}

	// If a single workspace block exceeds the viewport, scroll inside that block.
	if len(cursorGroup.lines) > maxVisible {
		start, end := listWindow(len(cursorGroup.lines), cursorLine, maxVisible)
		for i := start; i < end; i++ {
			b.WriteString(cursorGroup.lines[i])
			b.WriteString("\n")
		}
		return
	}

	startGroup, endGroup := groupWindowByLineBudget(groups, cursorWorkspace, maxVisible)
	for gi := startGroup; gi < endGroup; gi++ {
		for _, line := range groups[gi].lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
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

	width := currentWrapWidth()
	groups := make([]workspaceRepoChoiceGroup, 0, len(items))
	for i := range items {
		item := items[i]
		connectorToken := mutedToken(theme, useColor, output.LogConnector)
		displayID := item.ID
		warnValue := shortWarningTag(item.Warning)
		hasWarn := strings.TrimSpace(warnValue) != "" && strings.TrimSpace(strings.ToLower(warnValue)) != "clean"
		warnStyle := theme.Warn
		if item.WarningStrong {
			warnStyle = theme.Error
		}
		warnTag := ""
		if hasWarn {
			warnTag = "[" + warnValue + "]"
		}
		if useColor && i == cursor {
			displayID = lipgloss.NewStyle().Bold(true).Render(displayID)
		}
		display := displayID
		if warnTag != "" {
			tag := warnTag
			if useColor {
				if hasWarn {
					tag = warnStyle.Render(warnTag)
				} else {
					tag = theme.Accent.Render(warnTag)
				}
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
		line := fmt.Sprintf("%s%s %s", output.Indent+output.Indent, connectorToken, display)
		groups = append(groups, workspaceRepoChoiceGroup{lines: wrapRawLineToWidth(line, width)})
	}

	if cursor < 0 || cursor >= len(groups) {
		cursor = 0
	}
	if maxVisible <= 0 {
		maxVisible = 1
	}
	if len(groups[cursor].lines) > maxVisible {
		start, end := listWindow(len(groups[cursor].lines), 0, maxVisible)
		for i := start; i < end; i++ {
			b.WriteString(groups[cursor].lines[i])
			b.WriteString("\n")
		}
		return
	}
	startGroup, endGroup := groupWindowByLineBudget(groups, cursor, maxVisible)
	for gi := startGroup; gi < endGroup; gi++ {
		for _, line := range groups[gi].lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
}

func renderWorkspaceChoiceConfirmList(b *strings.Builder, items []WorkspaceChoice, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	for i, item := range items {
		workspaceConnector := "├─"
		isLastWorkspace := i == len(items)-1
		if isLastWorkspace {
			workspaceConnector = "└─"
		}
		connectorToken := workspaceConnector
		if useColor {
			connectorToken = theme.Muted.Render(workspaceConnector)
		}
		workspaceStem := "│ "
		if isLastWorkspace {
			workspaceStem = "  "
		}

		displayID := item.ID
		warnValue := shortWarningTag(item.Warning)
		hasWarn := strings.TrimSpace(warnValue) != "" && strings.TrimSpace(strings.ToLower(warnValue)) != "clean"
		warnStyle := theme.Warn
		if item.WarningStrong {
			warnStyle = theme.Error
		}
		warnTag := ""
		if hasWarn {
			warnTag = "[" + warnValue + "]"
		}
		display := displayID
		if warnTag != "" {
			tag := warnTag
			if useColor {
				if hasWarn {
					tag = warnStyle.Render(warnTag)
				} else {
					tag = theme.Accent.Render(warnTag)
				}
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
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, connectorToken, display))
		for j, repo := range item.Repos {
			repoConnector := "├─"
			isLastRepo := j == len(item.Repos)-1
			if isLastRepo {
				repoConnector = "└─"
			}
			line := fmt.Sprintf("%s%s%s %s", output.Indent+output.Indent, workspaceStem, repoConnector, repo.Label)
			if useColor {
				line = theme.Muted.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
			if len(repo.Details) == 0 {
				continue
			}
			repoStem := "│  "
			if isLastRepo {
				repoStem = "   "
			}
			for _, detail := range repo.Details {
				if strings.TrimSpace(detail) == "" {
					continue
				}
				detailLine := fmt.Sprintf("%s%s%s%s", output.Indent+output.Indent, workspaceStem, repoStem, detail)
				if useColor {
					detailLine = theme.Muted.Render(detailLine)
				}
				b.WriteString(detailLine)
				b.WriteString("\n")
			}
		}
	}
}

func shortWarningTag(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "dirty changes":
		return "dirty"
	case "unpushed commits":
		return "unpushed"
	case "diverged or upstream missing":
		return "diverged"
	case "status unknown":
		return "unknown"
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
