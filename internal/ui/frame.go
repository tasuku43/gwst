package ui

import (
	"io"
	"strings"
)

// Frame renders a single-screen layout with fixed section order.
type Frame struct {
	Inputs     []frameLine
	Info       []frameLine
	Steps      []frameLine
	Result     []frameLine
	Suggestion []frameLine

	theme    Theme
	useColor bool
}

func NewFrame(theme Theme, useColor bool) *Frame {
	return &Frame{theme: theme, useColor: useColor}
}

func (f *Frame) SetInputs(lines ...string) {
	f.Inputs = copyLines(lines, lineBullet)
}

func (f *Frame) SetInputsPrompt(lines ...string) {
	f.Inputs = copyLines(lines, linePrompt)
}

func (f *Frame) AppendInputsPrompt(lines ...string) {
	f.Inputs = append(f.Inputs, copyLines(lines, linePrompt)...)
}

func (f *Frame) SetInputsRaw(lines ...string) {
	f.Inputs = copyRawLines(lines)
}

func (f *Frame) AppendInputsRaw(lines ...string) {
	f.Inputs = append(f.Inputs, copyRawLines(lines)...)
}

func (f *Frame) SetInfo(lines ...string) {
	f.Info = copyLines(lines, lineBullet)
}

func (f *Frame) AppendInfo(lines ...string) {
	f.Info = append(f.Info, copyLines(lines, lineBullet)...)
}

func (f *Frame) SetInfoRaw(lines ...string) {
	f.Info = copyRawLines(lines)
}

func (f *Frame) AppendInfoRaw(lines ...string) {
	f.Info = append(f.Info, copyRawLines(lines)...)
}

func (f *Frame) SetSteps(lines ...string) {
	f.Steps = copyLines(lines, lineStep)
}

func (f *Frame) AppendSteps(lines ...string) {
	f.Steps = append(f.Steps, copyLines(lines, lineStep)...)
}

func (f *Frame) SetResult(lines ...string) {
	f.Result = copyLines(lines, lineBullet)
}

func (f *Frame) AppendResult(lines ...string) {
	f.Result = append(f.Result, copyLines(lines, lineBullet)...)
}

func (f *Frame) SetSuggestion(lines ...string) {
	f.Suggestion = copyLines(lines, lineBullet)
}

func (f *Frame) Render() string {
	var b strings.Builder
	_, _ = f.WriteTo(&b)
	return b.String()
}

func (f *Frame) WriteTo(w io.Writer) (int64, error) {
	cw := &countingWriter{w: w}
	r := NewRenderer(cw, f.theme, f.useColor)

	if len(f.Inputs) > 0 {
		r.Section("Inputs")
		for _, line := range f.Inputs {
			renderLine(r, line)
		}
		r.Blank()
	}

	if len(f.Info) > 0 {
		r.Section("Info")
		for _, line := range f.Info {
			renderLine(r, line)
		}
		r.Blank()
	}

	if len(f.Steps) > 0 {
		r.Section("Steps")
		for _, line := range f.Steps {
			renderLine(r, line)
		}
		r.Blank()
	}

	if len(f.Result) > 0 {
		r.Section("Result")
		for _, line := range f.Result {
			renderLine(r, line)
		}
		r.Blank()
	}

	if len(f.Suggestion) > 0 && f.useColor {
		r.Section("Suggestion")
		for _, line := range f.Suggestion {
			renderLine(r, line)
		}
	}

	return cw.n, cw.err
}

type frameLine struct {
	text string
	kind frameLineKind
}

type frameLineKind int

const (
	lineBullet frameLineKind = iota
	linePrompt
	lineStep
	lineRaw
)

type countingWriter struct {
	w   io.Writer
	n   int64
	err error
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	if cw.err != nil {
		return 0, cw.err
	}
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	if err != nil {
		cw.err = err
	}
	return n, err
}

func renderLine(r *Renderer, line frameLine) {
	if strings.TrimSpace(line.text) == "" {
		return
	}
	switch line.kind {
	case lineRaw:
		r.LineRaw(line.text)
	case linePrompt:
		r.Prompt(line.text)
	case lineStep:
		r.Step(line.text)
	default:
		r.Bullet(line.text)
	}
}

func copyLines(lines []string, kind frameLineKind) []frameLine {
	var out []frameLine
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, frameLine{text: trimmed, kind: kind})
	}
	return out
}

func copyRawLines(lines []string) []frameLine {
	var out []frameLine
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\n")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		out = append(out, frameLine{text: trimmed, kind: lineRaw})
	}
	return out
}
