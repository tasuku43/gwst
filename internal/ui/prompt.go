package ui

import (
	"errors"
	"fmt"
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

type WorkspaceChoice struct {
	ID    string
	Repos []PromptChoice
}

type BlockedChoice struct {
	Label string
}

func PromptNewWorkspaceInputs(title string, templates []string, templateName string, workspaceID string, theme Theme, useColor bool) (string, string, error) {
	model := newInputsModel(title, templates, templateName, workspaceID, theme, useColor)
	prog := tea.NewProgram(model)
	out, err := prog.Run()
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
	prog := tea.NewProgram(model)
	out, err := prog.Run()
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
	prog := tea.NewProgram(model)
	out, err := prog.Run()
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
	prog := tea.NewProgram(model)
	out, err := prog.Run()
	if err != nil {
		return "", err
	}
	final := out.(workspaceSelectModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.workspaceID), nil
}

func PromptConfirmInline(label string, theme Theme, useColor bool) (bool, error) {
	model := newConfirmInlineModel(label, theme, useColor)
	prog := tea.NewProgram(model)
	out, err := prog.Run()
	if err != nil {
		return false, err
	}
	final := out.(confirmInlineModel)
	if final.err != nil {
		return false, final.err
	}
	return final.value, nil
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

	stage     inputsStage
	theme     Theme
	useColor  bool
	search    textinput.Model
	idInput   textinput.Model
	filtered  []string
	cursor    int
	err       error
	errorLine string
}

func newInputsModel(title string, templates []string, templateName string, workspaceID string, theme Theme, useColor bool) inputsModel {
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
	var b strings.Builder
	header := formatInputsHeader(m.title, m.template, m.currentWorkspaceID())
	if m.useColor {
		header = m.theme.Header.Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n\n")
	inputsTitle := "Inputs"
	if m.useColor {
		inputsTitle = m.theme.SectionTitle.Render(inputsTitle)
	}
	b.WriteString(inputsTitle)
	b.WriteString("\n")

	if m.stage == stageTemplate {
		prefix := promptPrefix(m.theme, m.useColor)
		label := promptLabel(m.theme, m.useColor, "template")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.search.View())
		b.WriteString(line)
		b.WriteString("\n")
		if len(m.filtered) == 0 {
			msg := "no matches"
			if m.useColor {
				msg = m.theme.Muted.Render(msg)
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
		} else {
			for i, item := range m.filtered {
				display := item
				if i == m.cursor && m.useColor {
					display = lipgloss.NewStyle().Bold(true).Render(display)
				}
				b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), display))
			}
		}
	} else {
		prefix := promptPrefix(m.theme, m.useColor)
		label := promptLabel(m.theme, m.useColor, "template")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.template)
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.stage == stageWorkspace {
		prefix := promptPrefix(m.theme, m.useColor)
		label := promptLabel(m.theme, m.useColor, "workspace id")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.idInput.View())
		b.WriteString(line)
		b.WriteString("\n")
		if m.errorLine != "" {
			msg := m.errorLine
			if m.useColor {
				msg = m.theme.Error.Render(msg)
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
		}
	} else if strings.TrimSpace(m.workspaceID) != "" {
		prefix := promptPrefix(m.theme, m.useColor)
		label := promptLabel(m.theme, m.useColor, "workspace id")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.workspaceID)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
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
	label    string
	theme    Theme
	useColor bool
	input    textinput.Model
	value    bool
	err      error
}

func newConfirmInlineModel(label string, theme Theme, useColor bool) confirmInlineModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "y/n"
	ti.Focus()
	if useColor {
		ti.PlaceholderStyle = theme.Muted
	}
	return confirmInlineModel{
		label:    label,
		theme:    theme,
		useColor: useColor,
		input:    ti,
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
				return m, tea.Quit
			case "n", "no":
				m.value = false
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
	prefix := promptPrefix(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, m.label)
	line := fmt.Sprintf("%s%s %s (y/n): %s", output.Indent, prefix, label, m.input.View())
	return line + "\n"
}

// PromptInputInline collects a single inline value with an optional default and validation.
// Empty input accepts the default. Validation errors are shown inline and reprompted.
func PromptInputInline(label, defaultValue string, validate func(string) error, theme Theme, useColor bool) (string, error) {
	model := newInputInlineModel(label, defaultValue, validate, theme, useColor)
	prog := tea.NewProgram(model)
	out, err := prog.Run()
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
		title:        "Input",
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
	var b strings.Builder
	title := strings.TrimSpace(m.title)
	if title == "" {
		title = "Input"
	}
	if m.useColor {
		title = m.theme.SectionTitle.Render(title)
	}
	b.WriteString(title)
	b.WriteString("\n")

	prefix := promptPrefix(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, m.label)
	defaultText := ""
	if strings.TrimSpace(m.defaultValue) != "" {
		defaultText = fmt.Sprintf(" [default: %s]", m.defaultValue)
	}
	line := fmt.Sprintf("%s%s %s%s: %s", output.Indent, prefix, label, defaultText, m.input.View())
	if strings.TrimSpace(m.errorLine) != "" {
		errLine := m.errorLine
		if m.useColor {
			errLine = m.theme.Error.Render(errLine)
		}
		line = fmt.Sprintf("%s\n%s%s%s %s", line, output.Indent, output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), errLine)
	}
	b.WriteString(line)
	b.WriteString("\n")
	return b.String()
}

// PromptTemplateRepos lets users pick one or more repos from a list with filtering.
// It can also collect a template name when not provided.
func PromptTemplateRepos(title string, templateName string, choices []PromptChoice, theme Theme, useColor bool) (string, []string, error) {
	model := newTemplateRepoSelectModel(title, templateName, choices, theme, useColor)
	prog := tea.NewProgram(model)
	out, err := prog.Run()
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
	prog := tea.NewProgram(model)
	out, err := prog.Run()
	if err != nil {
		return "", err
	}
	final := out.(templateNameModel)
	if final.err != nil {
		return "", final.err
	}
	return strings.TrimSpace(final.value), nil
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
	var b strings.Builder
	section := "Input"
	if m.useColor {
		section = m.theme.SectionTitle.Render(section)
	}
	b.WriteString(section)
	b.WriteString("\n")

	prefix := promptPrefix(m.theme, m.useColor)
	labelName := promptLabel(m.theme, m.useColor, "template name")
	templateName := m.templateName
	if m.stage == stageTemplateName {
		templateName = m.nameInput.View()
	}
	lineName := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, labelName, templateName)
	b.WriteString(lineName)
	b.WriteString("\n")

	label := promptLabel(m.theme, m.useColor, "repo")
	repoInput := m.repoInput.View()
	if m.stage == stageTemplateName {
		repoInput = ""
	}
	line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, repoInput)
	b.WriteString(line)
	b.WriteString("\n")
	renderRepoChoiceList(&b, m.filtered, m.cursor, m.useColor, m.theme)

	b.WriteString("\n")
	selTitle := "Selected"
	if m.useColor {
		selTitle = m.theme.SectionTitle.Render(selTitle)
	}
	b.WriteString(selTitle)
	b.WriteString("\n")
	renderSelectedRepoList(&b, m.selected, m.useColor, m.theme)

	if m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}

	infoPrefix := mutedToken(m.theme, m.useColor, output.StepPrefix)
	b.WriteString(fmt.Sprintf("\n%s%s finish: Ctrl+D or type \"done\"\n", output.Indent, infoPrefix))
	b.WriteString(fmt.Sprintf("%s%s enter: add highlighted repo\n", output.Indent, infoPrefix))
	return b.String()
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
	var b strings.Builder
	header := m.title
	if strings.TrimSpace(m.value) != "" {
		header = fmt.Sprintf("%s (template: %s)", m.title, m.value)
	}
	if m.useColor {
		header = m.theme.Header.Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	section := "Input"
	if m.useColor {
		section = m.theme.SectionTitle.Render(section)
	}
	b.WriteString(section)
	b.WriteString("\n")

	prefix := promptPrefix(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "template name")
	line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.input.View())
	b.WriteString(line)
	b.WriteString("\n")

	if m.errorLine != "" {
		msg := m.errorLine
		if m.useColor {
			msg = m.theme.Error.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(m.theme, m.useColor, output.LogConnector), msg))
	}
	return b.String()
}

func promptPrefix(theme Theme, useColor bool) string {
	prefix := output.StepPrefix
	if useColor {
		return theme.Accent.Render(prefix)
	}
	return prefix
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	var b strings.Builder
	header := m.title
	if strings.TrimSpace(m.workspaceID) != "" {
		header = fmt.Sprintf("%s (workspace id: %s)", m.title, m.workspaceID)
	}
	if m.useColor {
		header = m.theme.Header.Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	title := "Inputs"
	if m.useColor {
		title = m.theme.SectionTitle.Render(title)
	}
	b.WriteString(title)
	b.WriteString("\n")

	prefix := promptPrefix(m.theme, m.useColor)
	label := promptLabel(m.theme, m.useColor, "workspace id")
	line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.input.View())
	b.WriteString(line)
	b.WriteString("\n")
	renderWorkspaceChoiceList(&b, m.filtered, m.cursor, m.useColor, m.theme)
	if len(m.blocked) > 0 {
		b.WriteString("\n")
		infoTitle := "Info"
		if m.useColor {
			infoTitle = m.theme.SectionTitle.Render(infoTitle)
		}
		b.WriteString(infoTitle)
		b.WriteString("\n")
		infoPrefix := output.StepPrefix
		if m.useColor {
			infoPrefix = m.theme.Muted.Render(infoPrefix)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent, infoPrefix, "blocked workspaces"))
		renderBlockedChoiceList(&b, m.blocked, m.useColor, m.theme)
	}
	return b.String()
}

func (m workspaceSelectModel) filterWorkspaces() []WorkspaceChoice {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if q == "" {
		return append([]WorkspaceChoice(nil), m.workspaces...)
	}
	var out []WorkspaceChoice
	for _, item := range m.workspaces {
		if strings.Contains(strings.ToLower(item.ID), q) {
			out = append(out, item)
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
	var b strings.Builder
	header := formatAddInputsHeader(m.title, m.workspaceID, m.currentRepoLabel())
	if m.useColor {
		header = m.theme.Header.Render(header)
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	title := "Inputs"
	if m.useColor {
		title = m.theme.SectionTitle.Render(title)
	}
	b.WriteString(title)
	b.WriteString("\n")

	prefix := promptPrefix(m.theme, m.useColor)

	if m.stage == addStageWorkspace {
		label := promptLabel(m.theme, m.useColor, "workspace id")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.wsInput.View())
		b.WriteString(line)
		b.WriteString("\n")
		renderWorkspaceChoiceList(&b, m.wsFiltered, m.cursor, m.useColor, m.theme)
	} else {
		label := promptLabel(m.theme, m.useColor, "workspace id")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.workspaceID)
		b.WriteString(line)
		b.WriteString("\n")
		if selected, ok := m.selectedWorkspace(); ok {
			renderRepoDetailList(&b, selected.Repos, output.Indent+output.Indent, m.useColor, m.theme)
		}
	}

	if m.stage == addStageRepo {
		label := promptLabel(m.theme, m.useColor, "repo")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.repoInput.View())
		b.WriteString(line)
		b.WriteString("\n")
		renderChoiceList(&b, m.repoLabels(), m.cursor, m.useColor, m.theme)
	} else if m.repoLabel != "" {
		label := promptLabel(m.theme, m.useColor, "repo")
		line := fmt.Sprintf("%s%s %s: %s", output.Indent, prefix, label, m.repoLabel)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
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
		if strings.Contains(strings.ToLower(item.ID), q) {
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

func renderChoiceList(b *strings.Builder, items []string, cursor int, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	for i, item := range items {
		display := item
		if i == cursor && useColor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), display))
	}
}

func renderRepoChoiceList(b *strings.Builder, items []PromptChoice, cursor int, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	for i, item := range items {
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

func renderWorkspaceChoiceList(b *strings.Builder, items []WorkspaceChoice, cursor int, useColor bool, theme Theme) {
	if len(items) == 0 {
		msg := "no matches"
		if useColor {
			msg = theme.Muted.Render(msg)
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", output.Indent+output.Indent, mutedToken(theme, useColor, output.LogConnector), msg))
		return
	}
	for i, item := range items {
		display := item.ID
		if i == cursor && useColor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
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
