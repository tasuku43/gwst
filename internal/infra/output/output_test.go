package output

import (
	"strings"
	"testing"
	"unicode/utf8"
)

type captureLogger struct {
	steps      []string
	logs       []string
	logOutputs []string
}

func (c *captureLogger) Step(text string) {
	c.steps = append(c.steps, text)
}

func (c *captureLogger) Log(text string) {
	c.logs = append(c.logs, text)
}

func (c *captureLogger) LogOutput(text string) {
	c.logOutputs = append(c.logOutputs, text)
}

func TestLogOutputPrefix(t *testing.T) {
	want := Indent + Indent + strings.Repeat(" ", utf8.RuneCountInString(LogConnector)+1)
	if got := LogOutputPrefix(); got != want {
		t.Fatalf("LogOutputPrefix() = %q, want %q", got, want)
	}
}

func TestLogLinesUsesLogOutput(t *testing.T) {
	logger := &captureLogger{}
	SetStepLogger(logger)
	defer SetStepLogger(nil)

	LogLines("alpha\nbravo\n")

	if len(logger.logOutputs) != 2 {
		t.Fatalf("logOutputs = %d, want 2", len(logger.logOutputs))
	}
	if len(logger.logs) != 0 {
		t.Fatalf("logs = %d, want 0", len(logger.logs))
	}
}
