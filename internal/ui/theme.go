package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Header       lipgloss.Style
	SectionTitle lipgloss.Style
	Success      lipgloss.Style
	Warn         lipgloss.Style
	Error        lipgloss.Style
	Muted        lipgloss.Style
}

func DefaultTheme() Theme {
	return Theme{
		Header:       lipgloss.NewStyle().Bold(true),
		SectionTitle: lipgloss.NewStyle().Bold(true),
		Success:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Warn:         lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		Error:        lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		Muted:        lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	}
}
