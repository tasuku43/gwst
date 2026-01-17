package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = "templates.yaml"

type File struct {
	Templates map[string]Template `yaml:"templates"`
}

type Template struct {
	Repos []string `yaml:"repos"`
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func (t *Template) UnmarshalYAML(value *yaml.Node) error {
	type rawTemplate struct {
		Repos []string `yaml:"repos"`
	}
	var direct rawTemplate
	if err := value.Decode(&direct); err == nil && len(direct.Repos) > 0 {
		t.Repos = direct.Repos
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
			t.Repos = append(t.Repos, item.Repo)
		}
		return nil
	}

	return value.Decode(&direct)
}

func Load(rootDir string) (File, error) {
	if rootDir == "" {
		return File{}, fmt.Errorf("root directory is required")
	}
	path := filepath.Join(rootDir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, fmt.Errorf("templates file not found: %s", path)
		}
		return File{}, err
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		return File{}, err
	}
	if file.Templates == nil {
		file.Templates = map[string]Template{}
	}
	return file, nil
}

func Names(file File) []string {
	var names []string
	for name := range file.Templates {
		if strings.TrimSpace(name) == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateName checks template name rules.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("template name is required")
	}
	if !namePattern.MatchString(trimmed) {
		return fmt.Errorf("invalid template name: %s", name)
	}
	return nil
}

// NormalizeRepos trims and de-duplicates repo specs while preserving order.
func NormalizeRepos(repos []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, repo := range repos {
		trimmed := strings.TrimSpace(repo)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

// Save writes the templates file atomically. It requires the file to already exist.
func Save(rootDir string, file File) error {
	if rootDir == "" {
		return fmt.Errorf("root directory is required")
	}
	path := filepath.Join(rootDir, FileName)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("templates file not found: %s", path)
		}
		return err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(file); err != nil {
		_ = enc.Close()
		return fmt.Errorf("marshal templates: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("close templates encoder: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "templates-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp templates file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp templates file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp templates file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace templates file: %w", err)
	}
	if err := os.Chmod(path, info.Mode()); err != nil {
		return fmt.Errorf("chmod templates file: %w", err)
	}
	return nil
}
