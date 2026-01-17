package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/tasuku43/gws/internal/core/debuglog"
	"github.com/tasuku43/gws/internal/core/output"
)

type Renderer struct {
	out       io.Writer
	theme     Theme
	useColor  bool
	wrapWidth int
}

func NewRenderer(out io.Writer, theme Theme, useColor bool) *Renderer {
	return &Renderer{
		out:       out,
		theme:     theme,
		useColor:  useColor,
		wrapWidth: currentWrapWidth(),
	}
}

func (r *Renderer) Header(text string) {
	r.writeLine(r.style(text, r.theme.Header))
}

func (r *Renderer) Blank() {
	fmt.Fprintln(r.out)
}

func (r *Renderer) Section(title string) {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "inputs":
		debuglog.SetPhase("inputs")
	case "info":
		debuglog.SetPhase("info")
	case "steps":
		debuglog.SetPhase("steps")
	case "result":
		debuglog.SetPhase("result")
	default:
		debuglog.SetPhase("none")
	}
	r.writeLine(r.style(title, r.theme.SectionTitle))
}

func (r *Renderer) Step(text string) {
	r.bullet(text)
}

func (r *Renderer) StepLog(text string) {
	r.writeWithPrefix(output.Indent+output.Indent+output.LogConnector+" ", r.style(text, r.theme.Muted))
}

func (r *Renderer) StepLogOutput(text string) {
	r.writeWithPrefix(output.LogOutputPrefix(), r.style(text, r.theme.Muted))
}

func (r *Renderer) Result(text string) {
	r.bullet(text)
}

func (r *Renderer) Bullet(text string) {
	r.bullet(text)
}

func (r *Renderer) Prompt(text string) {
	prefix := output.StepPrefix + " "
	if r.useColor {
		prefix = r.theme.Accent.Render(output.StepPrefix) + " "
	}
	r.writeWithPrefix(output.Indent+prefix, text)
}

func (r *Renderer) BulletWithDescription(id, description, suffix string) {
	prefix := output.StepPrefix + " "
	if r.useColor {
		prefix = r.theme.Muted.Render(prefix)
	}
	line := id
	desc := strings.TrimSpace(description)
	if desc != "" {
		if r.useColor {
			line += r.theme.Muted.Render(" - " + desc)
		} else {
			line += " - " + desc
		}
	}
	if strings.TrimSpace(suffix) != "" {
		value := " " + strings.TrimSpace(suffix)
		if r.useColor {
			value = r.theme.Muted.Render(value)
		}
		line += value
	}
	r.writeWithPrefix(output.Indent+prefix, line)
}

func (r *Renderer) BulletError(text string) {
	prefix := output.StepPrefix + " "
	if r.useColor {
		prefix = r.theme.Error.Render(prefix)
		text = r.theme.Error.Render(text)
	}
	r.writeWithPrefix(output.Indent+prefix, text)
}

func (r *Renderer) Warn(text string) {
	r.writeWithPrefix(output.Indent, r.style(text, r.theme.Warn))
}

func (r *Renderer) TreeLine(prefix, name string) {
	r.writeWithPrefix(output.Indent+prefix, name)
}

func (r *Renderer) TreeLineBranch(prefix, name, branch string) {
	line := name
	if strings.TrimSpace(branch) != "" {
		suffix := fmt.Sprintf(" (branch: %s)", branch)
		if r.useColor {
			suffix = r.style(suffix, r.theme.Accent)
		}
		line += suffix
	}
	r.writeWithPrefix(output.Indent+prefix, line)
}

func (r *Renderer) TreeLineBranchMuted(prefix, name, branch string) {
	line := name
	if strings.TrimSpace(branch) != "" {
		line += fmt.Sprintf(" (branch: %s)", branch)
	}
	fullPrefix := output.Indent + prefix
	if r.useColor {
		fullPrefix = r.style(fullPrefix, r.theme.Muted)
		line = r.style(line, r.theme.Muted)
	}
	r.writeWithPrefix(fullPrefix, line)
}

func (r *Renderer) TreeLineWarn(prefix, text string) {
	line := text
	fullPrefix := output.Indent + prefix
	if r.useColor {
		fullPrefix = r.style(fullPrefix, r.theme.Warn)
		line = r.style(line, r.theme.Warn)
	}
	r.writeWithPrefix(fullPrefix, line)
}

func (r *Renderer) TreeLineError(prefix, text string) {
	line := text
	fullPrefix := output.Indent + prefix
	if r.useColor {
		fullPrefix = r.style(fullPrefix, r.theme.Error)
		line = r.style(line, r.theme.Error)
	}
	r.writeWithPrefix(fullPrefix, line)
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
	r.writeWithPrefix(output.Indent+prefix, text)
}

func (r *Renderer) writeWithPrefix(prefix, text string) {
	if r.wrapWidth <= 0 {
		r.writeLine(prefix + text)
		return
	}
	prefixWidth := lipgloss.Width(prefix)
	available := r.wrapWidth - prefixWidth
	if available <= 0 {
		r.writeLine(prefix + text)
		return
	}
	wrapped := ansi.Wrap(text, available, "")
	lines := strings.Split(wrapped, "\n")
	if len(lines) == 0 {
		return
	}
	r.writeLine(prefix + lines[0])
	if len(lines) == 1 {
		return
	}
	padding := strings.Repeat(" ", prefixWidth)
	for _, line := range lines[1:] {
		r.writeLine(padding + line)
	}
}

func (r *Renderer) writeLine(text string) {
	fmt.Fprintln(r.out, strings.TrimRight(text, "\n"))
}

func (r *Renderer) LineRaw(text string) {
	prefix, rest, ok := splitRawLinePrefix(text)
	if ok {
		r.writeWithPrefix(prefix, rest)
		return
	}
	r.writeWithPrefix("", text)
}

func (r *Renderer) Log(text string) {
	r.StepLog(text)
}

func (r *Renderer) LogOutput(text string) {
	r.StepLogOutput(text)
}

func splitRawLinePrefix(line string) (string, string, bool) {
	plain := ansi.Strip(line)
	if strings.TrimSpace(plain) == "" {
		return "", "", false
	}
	runes := []rune(plain)
	for i := 1; i < len(runes)-1; i++ {
		if runes[i] != ' ' {
			continue
		}
		if runes[i-1] == ' ' || runes[i+1] == ' ' {
			continue
		}
		prefixPlain := string(runes[:i+1])
		prefixWidth := ansi.StringWidth(prefixPlain)
		if prefixWidth <= 0 {
			break
		}
		totalWidth := ansi.StringWidth(line)
		if totalWidth <= prefixWidth {
			break
		}
		prefix := ansi.Cut(line, 0, prefixWidth)
		rest := ansi.Cut(line, prefixWidth, totalWidth)
		return prefix, rest, true
	}
	return "", "", false
}
