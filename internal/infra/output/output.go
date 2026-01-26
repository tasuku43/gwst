package output

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/tasuku43/gion/internal/infra/debuglog"
)

const (
	Indent       = "  "
	Indent2      = Indent + Indent
	Indent3      = Indent2 + Indent
	StepPrefix   = "•"
	LogConnector = "└─"

	TreeBranchMid  = "├─ "
	TreeBranchLast = "└─ "
	TreeStemMid    = "│  "
	TreeStemLast   = "   "
)

type StepLogger interface {
	Step(text string)
	Log(text string)
	LogOutput(text string)
}

var stepLogger StepLogger
var stepIndex uint64

func SetStepLogger(logger StepLogger) {
	stepLogger = logger
}

func HasStepLogger() bool {
	return stepLogger != nil
}

func Step(text string) {
	value := atomic.AddUint64(&stepIndex, 1)
	debuglog.SetStep(value, stepID(text))
	if stepLogger != nil {
		stepLogger.Step(text)
		return
	}
	fmt.Fprintf(os.Stdout, "%s%s %s\n", Indent, StepPrefix, text)
}

func Log(text string) {
	if stepLogger != nil {
		stepLogger.Log(text)
		return
	}
	fmt.Fprintf(os.Stdout, "%s%s %s\n", Indent+Indent, LogConnector, text)
}

func Logf(format string, args ...any) {
	Log(fmt.Sprintf(format, args...))
}

func LogOutput(text string) {
	if stepLogger != nil {
		stepLogger.LogOutput(text)
		return
	}
	fmt.Fprintf(os.Stdout, "%s%s\n", LogOutputPrefix(), text)
}

func LogOutputf(format string, args ...any) {
	LogOutput(fmt.Sprintf(format, args...))
}

func LogLines(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		LogOutput(line)
	}
}

func LogOutputPrefix() string {
	spaces := utf8.RuneCountInString(LogConnector) + 1
	return Indent + Indent + strings.Repeat(" ", spaces)
}

func stepID(text string) string {
	trimmed := strings.ToLower(strings.TrimSpace(text))
	if trimmed == "" {
		return "step"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "step"
	}
	if len(out) > 32 {
		return out[:32]
	}
	return out
}
