package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/tasuku43/gws/internal/core/output"
	"github.com/tasuku43/gws/internal/domain/workspace"
	"github.com/tasuku43/gws/internal/ui"
)

func runWorkspaceOpen(ctx context.Context, rootDir string, args []string, noPrompt bool) error {
	openFlags := flag.NewFlagSet("open", flag.ContinueOnError)
	var helpFlag bool
	var shellFlag bool
	openFlags.BoolVar(&shellFlag, "shell", false, "spawn interactive shell")
	openFlags.BoolVar(&helpFlag, "help", false, "show help")
	openFlags.BoolVar(&helpFlag, "h", false, "show help")
	openFlags.SetOutput(os.Stdout)
	openFlags.Usage = func() {
		printOpenHelp(os.Stdout)
	}
	if err := openFlags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if helpFlag {
		printOpenHelp(os.Stdout)
		return nil
	}
	if openFlags.NArg() > 1 {
		return fmt.Errorf("usage: gws open [<WORKSPACE_ID>] [--shell]")
	}

	if current := nestedOpenWorkspaceID(); current != "" {
		return fmt.Errorf("already in gws open workspace: %s (exit the subshell to switch)", current)
	}

	workspaceID := ""
	if openFlags.NArg() == 1 {
		workspaceID = openFlags.Arg(0)
	}

	if workspaceID == "" {
		if noPrompt {
			return fmt.Errorf("workspace id is required without prompt")
		}
		workspaces, wsWarn, err := workspace.List(rootDir)
		if err != nil {
			return err
		}
		if len(wsWarn) > 0 {
			// ignore warnings for selection
		}
		workspaceChoices := buildWorkspaceChoices(ctx, workspaces)
		if len(workspaceChoices) == 0 {
			return fmt.Errorf("no workspaces found")
		}
		theme := ui.DefaultTheme()
		useColor := isatty.IsTerminal(os.Stdout.Fd())
		workspaceID, err = ui.PromptWorkspace("gws open", workspaceChoices, theme, useColor)
		if err != nil {
			return err
		}
	}

	wsDir := filepath.Join(rootDir, "workspaces", workspaceID)
	if info, err := os.Stat(wsDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("workspace does not exist: %s", wsDir)
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("workspace path is not a directory: %s", wsDir)
	}

	shellPath := strings.TrimSpace(os.Getenv("SHELL"))
	cmdPath, cmdArgs := shellCommandForOpen(shellPath)
	cmdDisplay := cmdPath
	if len(cmdArgs) > 0 {
		cmdDisplay = fmt.Sprintf("%s %s", cmdPath, strings.Join(cmdArgs, " "))
	}
	theme := ui.DefaultTheme()
	useColor := isatty.IsTerminal(os.Stdout.Fd())
	renderer := ui.NewRenderer(os.Stdout, theme, useColor)
	output.SetStepLogger(renderer)
	defer output.SetStepLogger(nil)

	renderer.Section("Info")
	renderer.Bullet("subshell; parent cwd unchanged")
	renderer.Blank()
	startSteps(renderer)
	output.Step("chdir")
	output.Log(wsDir)
	output.Step("launch subshell")
	output.Log(cmdDisplay)
	renderer.Blank()
	renderer.Section("Result")
	renderer.Bullet("enter subshell (type `exit` to return)")
	if err := os.Chdir(wsDir); err != nil {
		return fmt.Errorf("chdir workspace: %w", err)
	}
	launchArgs := cmdArgs
	extraEnv := []string{}
	cleanup := func() {}
	if override, err := preparePromptOverride(cmdPath, workspaceID); err != nil {
		return err
	} else if override != nil {
		launchArgs = override.args
		extraEnv = override.env
		cleanup = override.cleanup
		cmdDisplay = fmt.Sprintf("%s %s", cmdPath, strings.Join(launchArgs, " "))
	}
	defer cleanup()
	cmd := exec.CommandContext(ctx, cmdPath, launchArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("GWS_WORKSPACE=%s", workspaceID))
	cmd.Env = append(cmd.Env, extraEnv...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open shell: %w", err)
	}
	return nil
}

func shellCommandForOpen(shellPath string) (string, []string) {
	if strings.TrimSpace(shellPath) == "" {
		shellPath = "/bin/sh"
	}
	name := filepath.Base(shellPath)
	if isInteractiveShell(name) {
		return shellPath, []string{"-i"}
	}
	return shellPath, nil
}

func nestedOpenWorkspaceID() string {
	return strings.TrimSpace(os.Getenv("GWS_WORKSPACE"))
}

func isInteractiveShell(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bash", "zsh", "sh", "fish", "ksh", "dash", "tcsh", "csh":
		return true
	default:
		return false
	}
}

type promptOverride struct {
	args    []string
	env     []string
	cleanup func()
}

func preparePromptOverride(shellPath string, workspaceID string) (*promptOverride, error) {
	name := strings.ToLower(strings.TrimSpace(filepath.Base(shellPath)))
	switch name {
	case "bash":
		return prepareBashPromptOverride(workspaceID)
	case "zsh":
		return prepareZshPromptOverride(workspaceID)
	case "sh":
		return prepareShPromptOverride(workspaceID)
	default:
		return nil, nil
	}
}

func prepareBashPromptOverride(workspaceID string) (*promptOverride, error) {
	dir, err := os.MkdirTemp("", "gws-open-bash-")
	if err != nil {
		return nil, fmt.Errorf("open shell: create temp dir: %w", err)
	}
	rcfile := filepath.Join(dir, "bashrc")
	home := os.Getenv("HOME")
	fallback := workspaceID
	content := strings.Join([]string{
		"# gws open prompt wrapper",
		fmt.Sprintf("if [ -r %s ]; then . %s; fi", strconv.Quote(filepath.Join(home, ".bashrc")), strconv.Quote(filepath.Join(home, ".bashrc"))),
		fmt.Sprintf("ws=${GWS_WORKSPACE:-%s}", strconv.Quote(fallback)),
		"prefix=\"\\[\\033[34m\\][gws:${ws}]\\[\\033[0m\\] \"",
		"if [ -n \"$PS1\" ]; then",
		"  PS1=\"${prefix}${PS1}\"",
		"else",
		"  PS1=\"${prefix}\"",
		"fi",
		"export PS1",
		"",
	}, "\n")
	if err := os.WriteFile(rcfile, []byte(content), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("open shell: write bashrc: %w", err)
	}
	return &promptOverride{
		args:    []string{"--rcfile", rcfile, "-i"},
		env:     nil,
		cleanup: func() { _ = os.RemoveAll(dir) },
	}, nil
}

func prepareZshPromptOverride(workspaceID string) (*promptOverride, error) {
	dir, err := os.MkdirTemp("", "gws-open-zsh-")
	if err != nil {
		return nil, fmt.Errorf("open shell: create temp dir: %w", err)
	}
	rcfile := filepath.Join(dir, ".zshrc")
	histfile := filepath.Join(dir, ".zsh_history")
	if err := os.WriteFile(histfile, []byte{}, 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("open shell: write zsh history: %w", err)
	}
	home := os.Getenv("HOME")
	origZdot := os.Getenv("ZDOTDIR")
	if origZdot == "" {
		origZdot = home
	}
	fallback := workspaceID
	content := strings.Join([]string{
		"# gws open prompt wrapper",
		fmt.Sprintf("orig=${GWS_ZDOTDIR_ORIG:-%s}", strconv.Quote(origZdot)),
		"orig_hist=${HISTFILE:-$orig/.zsh_history}",
		"if [ -r \"$orig_hist\" ]; then",
		"  HISTFILE=\"$orig_hist\"",
		"else",
		"  HISTFILE=\"$GWS_HISTFILE_TMP\"",
		"fi",
		"export HISTFILE",
		"if [ -r \"$orig/.zshenv\" ]; then . \"$orig/.zshenv\"; fi",
		"if [ -r \"$orig/.zshrc\" ]; then . \"$orig/.zshrc\"; fi",
		fmt.Sprintf("ws=${GWS_WORKSPACE:-%s}", strconv.Quote(fallback)),
		"prefix=\"%F{blue}[gws:${ws}]%f \"",
		"if [ -n \"$PROMPT\" ]; then",
		"  if [[ \"$PROMPT\" == $'\\n'* ]]; then",
		"    PROMPT=$'\\n'\"${prefix}${PROMPT#$'\\n'}\"",
		"  else",
		"    PROMPT=\"${prefix}${PROMPT}\"",
		"  fi",
		"else",
		"  PROMPT=\"${prefix}\"",
		"fi",
		"",
	}, "\n")
	if err := os.WriteFile(rcfile, []byte(content), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("open shell: write zshrc: %w", err)
	}
	return &promptOverride{
		args: []string{"-i"},
		env: []string{
			fmt.Sprintf("ZDOTDIR=%s", dir),
			fmt.Sprintf("GWS_ZDOTDIR_ORIG=%s", origZdot),
			fmt.Sprintf("GWS_HISTFILE_TMP=%s", histfile),
		},
		cleanup: func() { _ = os.RemoveAll(dir) },
	}, nil
}

func prepareShPromptOverride(workspaceID string) (*promptOverride, error) {
	dir, err := os.MkdirTemp("", "gws-open-sh-")
	if err != nil {
		return nil, fmt.Errorf("open shell: create temp dir: %w", err)
	}
	rcfile := filepath.Join(dir, "shrc")
	origEnv := os.Getenv("ENV")
	fallback := workspaceID
	content := strings.Join([]string{
		"# gws open prompt wrapper",
		fmt.Sprintf("orig=${GWS_ENV_ORIG:-%s}", strconv.Quote(origEnv)),
		"if [ -n \"$orig\" ] && [ -r \"$orig\" ]; then . \"$orig\"; fi",
		fmt.Sprintf("ws=${GWS_WORKSPACE:-%s}", strconv.Quote(fallback)),
		"prefix=\"\\033[34m[gws:${ws}]\\033[0m \"",
		"if [ -n \"$PS1\" ]; then",
		"  PS1=\"${prefix}${PS1}\"",
		"else",
		"  PS1=\"${prefix}\"",
		"fi",
		"export PS1",
		"",
	}, "\n")
	if err := os.WriteFile(rcfile, []byte(content), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("open shell: write shrc: %w", err)
	}
	return &promptOverride{
		args: []string{"-i"},
		env: []string{
			fmt.Sprintf("ENV=%s", rcfile),
			fmt.Sprintf("GWS_ENV_ORIG=%s", origEnv),
		},
		cleanup: func() { _ = os.RemoveAll(dir) },
	}, nil
}
