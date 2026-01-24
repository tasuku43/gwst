package paths

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultRootDir = "gion"

func ResolveRoot(flagRoot string) (string, error) {
	if flagRoot != "" {
		return normalizeRoot(flagRoot)
	}

	envRoot := os.Getenv("GION_ROOT")
	if envRoot != "" {
		return normalizeRoot(envRoot)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultRootDir), nil
}

func normalizeRoot(path string) (string, error) {
	expanded, err := expandHome(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(expanded), nil
}

func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}

	return path, nil
}
