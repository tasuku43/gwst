package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Header       lipgloss.Style
	SectionTitle lipgloss.Style
	Success      lipgloss.Style
	Warn         lipgloss.Style
	SoftWarn     lipgloss.Style
	Error        lipgloss.Style
	Muted        lipgloss.Style
	Accent       lipgloss.Style
}

func DefaultTheme() Theme {
	return Theme{
		Header:       lipgloss.NewStyle().Bold(true),
		SectionTitle: lipgloss.NewStyle().Bold(true),
		Success:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Warn:         lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		SoftWarn:     lipgloss.NewStyle().Foreground(lipgloss.Color("180")),
		Error:        lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		Muted:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Accent:       lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	}
}
