package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tasuku43/gws/internal/output"
)

type Renderer struct {
	out      io.Writer
	theme    Theme
	useColor bool
}

func NewRenderer(out io.Writer, theme Theme, useColor bool) *Renderer {
	return &Renderer{
		out:      out,
		theme:    theme,
		useColor: useColor,
	}
}

func (r *Renderer) Header(text string) {
	r.writeLine(r.style(text, r.theme.Header))
}

func (r *Renderer) Blank() {
	fmt.Fprintln(r.out)
}

func (r *Renderer) Section(title string) {
	r.writeLine(r.style(title, r.theme.SectionTitle))
}

func (r *Renderer) Step(text string) {
	r.bullet(text)
}

func (r *Renderer) StepLog(text string) {
	r.writeLine(output.Indent + output.Indent + output.LogConnector + " " + r.style(text, r.theme.Muted))
}

func (r *Renderer) StepLogOutput(text string) {
	r.writeLine(output.LogOutputPrefix() + r.style(text, r.theme.Muted))
}

func (r *Renderer) Result(text string) {
	r.writeLine(output.Indent + text)
}

func (r *Renderer) Bullet(text string) {
	r.bullet(text)
}

func (r *Renderer) BulletError(text string) {
	prefix := output.StepPrefix + " "
	if r.useColor {
		prefix = r.theme.Error.Render(prefix)
		text = r.theme.Error.Render(text)
	}
	r.writeLine(output.Indent + prefix + text)
}

func (r *Renderer) Warn(text string) {
	r.writeLine(output.Indent + r.style(text, r.theme.Warn))
}

func (r *Renderer) TreeLine(prefix, name string) {
	r.writeLine(output.Indent + prefix + name)
}

func (r *Renderer) TreeLineBranch(prefix, name, branch string) {
	line := output.Indent + prefix + name
	if strings.TrimSpace(branch) != "" {
		suffix := fmt.Sprintf(" (branch: %s)", branch)
		if r.useColor {
			suffix = r.style(suffix, r.theme.Accent)
		}
		line += suffix
	}
	r.writeLine(line)
}

func (r *Renderer) TreeLineBranchMuted(prefix, name, branch string) {
	line := output.Indent + prefix + name
	if strings.TrimSpace(branch) != "" {
		line += fmt.Sprintf(" (branch: %s)", branch)
	}
	if r.useColor {
		line = r.style(line, r.theme.Muted)
	}
	r.writeLine(line)
}

func (r *Renderer) TreeLineWarn(prefix, text string) {
	line := output.Indent + prefix + text
	if r.useColor {
		line = r.style(line, r.theme.Warn)
	}
	r.writeLine(line)
}

func (r *Renderer) TreeLineError(prefix, text string) {
	line := output.Indent + prefix + text
	if r.useColor {
		line = r.style(line, r.theme.Error)
	}
	r.writeLine(line)
}

func (r *Renderer) style(text string, style lipgloss.Style) string {
	if !r.useColor {
		return text
	}
	return style.Render(text)
}

func (r *Renderer) bullet(text string) {
	prefix := output.StepPrefix + " "
	if r.useColor {
		prefix = r.theme.Muted.Render(prefix)
	}
	r.writeLine(output.Indent + prefix + text)
}

func (r *Renderer) writeLine(text string) {
	fmt.Fprintln(r.out, strings.TrimRight(text, "\n"))
}

func (r *Renderer) Log(text string) {
	r.StepLog(text)
}

func (r *Renderer) LogOutput(text string) {
	r.StepLogOutput(text)
}
