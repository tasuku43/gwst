package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tasuku43/gws/internal/output"
)

var ErrPromptCanceled = errors.New("prompt canceled")

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
