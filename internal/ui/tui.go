package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Header  string
	Steps   []string
	Results []string

	Theme Theme

	// Placeholder components for future interactive flows.
	List  list.Model
	Input textinput.Model
}

func NewModel(header string, theme Theme) Model {
	ti := textinput.New()
	ti.Prompt = ""
	return Model{
		Header: header,
		Theme:  theme,
		Input:  ti,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder
	r := NewRenderer(&b, m.Theme, true)
	if m.Header != "" {
		r.Header(m.Header)
		r.Blank()
	}
	if len(m.Steps) > 0 {
		r.Section("Steps")
		for _, step := range m.Steps {
			r.Step(step)
		}
		r.Blank()
	}
	if len(m.Results) > 0 {
		r.Section("Result")
		for _, line := range m.Results {
			r.Result(line)
		}
	}
	return b.String()
}
