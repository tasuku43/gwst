package manifest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = "gion.yaml"

type File struct {
	Version    int                  `yaml:"version"`
	Workspaces map[string]Workspace `yaml:"workspaces"`
	Presets    map[string]Preset    `yaml:"presets"`
}

type Workspace struct {
	Description string `yaml:"description,omitempty"`
	Mode        string `yaml:"mode,omitempty"`
	PresetName  string `yaml:"preset_name,omitempty"`
	SourceURL   string `yaml:"source_url,omitempty"`
	Repos       []Repo `yaml:"repos"`
}

type Preset struct {
	Repos []string `yaml:"repos"`
}

func (p *Preset) UnmarshalYAML(value *yaml.Node) error {
	type rawPreset struct {
		Repos []string `yaml:"repos"`
	}
	var direct rawPreset
	if err := value.Decode(&direct); err == nil && len(direct.Repos) > 0 {
		p.Repos = direct.Repos
		return nil
	}

	var legacy struct {
		Repos []struct {
			Repo string `yaml:"repo"`
		} `yaml:"repos"`
	}
	if err := value.Decode(&legacy); err == nil && len(legacy.Repos) > 0 {
		for _, item := range legacy.Repos {
			if strings.TrimSpace(item.Repo) == "" {
				continue
			}
			p.Repos = append(p.Repos, item.Repo)
		}
		return nil
	}

	return value.Decode(&direct)
}

type Repo struct {
	Alias   string `yaml:"alias"`
	RepoKey string `yaml:"repo_key"`
	Branch  string `yaml:"branch"`
	BaseRef string `yaml:"base_ref,omitempty"`
}

func Path(rootDir string) string {
	return filepath.Join(rootDir, FileName)
}

func Load(rootDir string) (File, error) {
	path := Path(rootDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", FileName, err)
	}
	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", FileName, err)
	}
	if file.Version == 0 {
		file.Version = 1
	}
	if file.Workspaces == nil {
		file.Workspaces = map[string]Workspace{}
	}
	if file.Presets == nil {
		file.Presets = map[string]Preset{}
	}
	return file, nil
}

func Save(rootDir string, file File) error {
	data, err := Marshal(file)
	if err != nil {
		return err
	}
	if err := os.WriteFile(Path(rootDir), data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}

func Marshal(file File) ([]byte, error) {
	if file.Version == 0 {
		file.Version = 1
	}
	if file.Workspaces == nil {
		file.Workspaces = map[string]Workspace{}
	}
	if file.Presets == nil {
		file.Presets = map[string]Preset{}
	}
	type rest struct {
		Presets    map[string]Preset    `yaml:"presets"`
		Workspaces map[string]Workspace `yaml:"workspaces"`
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(rest{Presets: file.Presets, Workspaces: file.Workspaces}); err != nil {
		_ = enc.Close()
		return nil, fmt.Errorf("marshal %s: %w", FileName, err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close %s encoder: %w", FileName, err)
	}
	out := []byte(fmt.Sprintf("version: %d\n\n%s", file.Version, buf.String()))
	return out, nil
}
