package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gion/internal/domain/manifest"
	"github.com/tasuku43/gion/internal/domain/preset"
)

func normalizeManifestAddArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--preset" || arg == "-preset" {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				out = append(out, arg+"=")
				continue
			}
		}
		if arg == "--repo" || arg == "-repo" {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				out = append(out, arg+"=")
				continue
			}
		}
		out = append(out, arg)
	}
	return out
}

func loadPresetNames(rootDir string) ([]string, error) {
	file, err := preset.Load(rootDir)
	if err != nil {
		return nil, err
	}
	names := preset.Names(file)
	if len(names) == 0 {
		return nil, fmt.Errorf("no presets found in %s", filepath.Join(rootDir, manifest.FileName))
	}
	return names, nil
}
