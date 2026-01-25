package debuglog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type loggerState struct {
	mu      sync.Mutex
	enabled atomic.Bool
	writer  *os.File
	pid     int
}

var state loggerState
var traceSeq uint64
var ctxState debugContext

type debugContext struct {
	mu     sync.Mutex
	phase  string
	prompt string
	step   string
	stepID string
}

func Enable(rootDir string) error {
	if strings.TrimSpace(rootDir) == "" {
		return fmt.Errorf("root directory is required")
	}
	logDir := filepath.Join(rootDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return fmt.Errorf("create debug log dir: %w", err)
	}
	name := fmt.Sprintf("debug-%s.log", time.Now().Format("20060102"))
	path := filepath.Join(logDir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open debug log file: %w", err)
	}
	state.mu.Lock()
	if state.writer != nil {
		_ = state.writer.Close()
	}
	state.writer = file
	state.pid = os.Getpid()
	state.enabled.Store(true)
	state.mu.Unlock()
	return nil
}

func Close() error {
	state.mu.Lock()
	state.enabled.Store(false)
	var err error
	if state.writer != nil {
		err = state.writer.Close()
		state.writer = nil
	}
	state.mu.Unlock()
	return err
}

func Enabled() bool {
	return state.enabled.Load()
}

func NewTrace(prefix string) string {
	value := atomic.AddUint64(&traceSeq, 1)
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "cmd"
	}
	return fmt.Sprintf("%s:%x", prefix, value)
}

func FormatCommand(name string, args []string) string {
	if len(args) == 0 {
		return strings.TrimSpace(name)
	}
	return strings.TrimSpace(name + " " + strings.Join(args, " "))
}

func LogCommand(trace, cmd string) {
	logLine(trace, "cmd", cmd, "", nil)
}

func LogStdoutLines(trace, text string) {
	logOutputLines(trace, "stdout", text)
}

func LogStderrLines(trace, text string) {
	logOutputLines(trace, "stderr", text)
}

func LogExit(trace string, code int) {
	logLine(trace, "exit", "", "", &code)
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	type exitCoder interface {
		ExitCode() int
	}
	if ec, ok := err.(exitCoder); ok {
		return ec.ExitCode()
	}
	return -1
}

func SetPrompt(label string) {
	ctxState.mu.Lock()
	ctxState.phase = "prompt"
	ctxState.prompt = strings.TrimSpace(label)
	ctxState.step = ""
	ctxState.stepID = ""
	ctxState.mu.Unlock()
}

func ClearPrompt() {
	ctxState.mu.Lock()
	if ctxState.phase == "prompt" {
		ctxState.phase = ""
	}
	ctxState.prompt = ""
	ctxState.mu.Unlock()
}

func SetStep(index uint64, stepID string) {
	ctxState.mu.Lock()
	ctxState.phase = "steps"
	ctxState.prompt = ""
	ctxState.step = fmt.Sprintf("%d", index)
	ctxState.stepID = strings.TrimSpace(stepID)
	ctxState.mu.Unlock()
}

func SetPhase(phase string) {
	ctxState.mu.Lock()
	ctxState.phase = strings.TrimSpace(phase)
	ctxState.prompt = ""
	ctxState.step = ""
	ctxState.stepID = ""
	ctxState.mu.Unlock()
}

func logOutputLines(trace, kind, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		logLine(trace, kind, "", line, nil)
	}
}

func logLine(trace, kind, cmd, line string, code *int) {
	if !Enabled() {
		return
	}
	trace = strings.TrimSpace(trace)
	if trace == "" {
		trace = "unknown"
	}
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "info"
	}
	phase, prompt, step, stepID := snapshotContext()
	if phase == "" {
		phase = "none"
	}
	ts := time.Now().Format(time.RFC3339Nano)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.writer == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ts=%s pid=%d trace=%s phase=%s kind=%s", ts, state.pid, trace, phase, kind)
	if prompt != "" {
		fmt.Fprintf(&b, " prompt=%q", prompt)
	}
	if step != "" {
		fmt.Fprintf(&b, " step=%s", step)
	}
	if stepID != "" {
		fmt.Fprintf(&b, " step_id=%s", stepID)
	}
	if cmd != "" {
		fmt.Fprintf(&b, " cmd=%q", cmd)
	}
	if line != "" {
		fmt.Fprintf(&b, " line=%q", line)
	}
	if code != nil {
		fmt.Fprintf(&b, " code=%d", *code)
	}
	b.WriteByte('\n')
	_, _ = state.writer.Write([]byte(b.String()))
}

func snapshotContext() (string, string, string, string) {
	ctxState.mu.Lock()
	defer ctxState.mu.Unlock()
	return ctxState.phase, ctxState.prompt, ctxState.step, ctxState.stepID
}
