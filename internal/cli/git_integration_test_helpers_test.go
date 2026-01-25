package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupLocalRemoteRepoExampleDotCom(t *testing.T, tmp string) (repoSpec string, remotePath string) {
	t.Helper()

	remoteBase := filepath.Join(tmp, "remotes")
	remotePath = filepath.Join(remoteBase, "example.com", "org", "repo.git")
	if err := os.MkdirAll(filepath.Dir(remotePath), 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, "", "init", "--bare", remotePath)

	seedDir := filepath.Join(tmp, "seed")
	runGit(t, "", "init", seedDir)
	runGit(t, seedDir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "init")
	runGit(t, seedDir, "remote", "add", "origin", remotePath)
	runGit(t, seedDir, "push", "origin", "main")
	runGit(t, "", "--git-dir", remotePath, "symbolic-ref", "HEAD", "refs/heads/main")

	// Keep tests offline: rewrite https://example.com/... to file://.../example.com/...
	configPath := filepath.Join(tmp, "gitconfig")
	fileURL := "file://" + filepath.ToSlash(remoteBase) + "/example.com/"
	configData := fmt.Sprintf("[url \"%s\"]\n\tinsteadOf = https://example.com/\n", fileURL)
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatalf("write gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	return "https://example.com/org/repo.git", remotePath
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s failed: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}
