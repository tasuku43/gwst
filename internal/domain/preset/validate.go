package preset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"gopkg.in/yaml.v3"
)

type ValidationIssue struct {
	Kind    string
	Preset  string
	Repo    string
	Message string
}

type ValidationResult struct {
	Path   string
	Issues []ValidationIssue
}

const (
	IssueKindFile              = "gwst.yaml"
	IssueKindInvalidYAML       = "invalid yaml"
	IssueKindMissingRequired   = "missing required field"
	IssueKindDuplicatePreset   = "duplicate preset name"
	IssueKindInvalidPresetName = "invalid preset name"
	IssueKindInvalidRepoSpec   = "invalid repo spec"
)

func Validate(rootDir string) (ValidationResult, error) {
	if strings.TrimSpace(rootDir) == "" {
		return ValidationResult{}, fmt.Errorf("root directory is required")
	}
	path := filepath.Join(rootDir, manifest.FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return ValidationResult{
			Path:   path,
			Issues: []ValidationIssue{joinFileIssue(err)},
		}, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return ValidationResult{
			Path:   path,
			Issues: []ValidationIssue{joinIssue(IssueKindInvalidYAML, "", "", err.Error())},
		}, nil
	}

	root := unwrapDocument(&doc)
	issues := validateRootPresets(root)
	return ValidationResult{Path: path, Issues: issues}, nil
}

func unwrapDocument(node *yaml.Node) *yaml.Node {
	if node == nil {
		return node
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func validateRootPresets(root *yaml.Node) []ValidationIssue {
	if root == nil || root.Kind != yaml.MappingNode {
		return []ValidationIssue{joinIssue(IssueKindMissingRequired, "", "", "presets")}
	}
	var presetsNode *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i]
		value := root.Content[i+1]
		if key != nil && key.Value == "presets" {
			presetsNode = value
			break
		}
	}
	if presetsNode == nil {
		return []ValidationIssue{joinIssue(IssueKindMissingRequired, "", "", "presets")}
	}
	if presetsNode.Kind != yaml.MappingNode {
		return []ValidationIssue{joinIssue(IssueKindMissingRequired, "", "", "presets must be a mapping")}
	}

	return validatePresetsMap(presetsNode)
}

func validatePresetsMap(node *yaml.Node) []ValidationIssue {
	var issues []ValidationIssue
	seen := make(map[string]struct{})

	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		name := ""
		if key != nil {
			name = strings.TrimSpace(key.Value)
		}
		if err := ValidateName(name); err != nil {
			issues = append(issues, joinIssue(IssueKindInvalidPresetName, name, "", err.Error()))
		}
		if name != "" {
			if _, ok := seen[name]; ok {
				issues = append(issues, joinIssue(IssueKindDuplicatePreset, name, "", "duplicate preset name"))
			} else {
				seen[name] = struct{}{}
			}
		}
		issues = append(issues, validatePresetEntry(name, value)...)
	}
	return issues
}

func validatePresetEntry(name string, node *yaml.Node) []ValidationIssue {
	if node == nil || node.Kind != yaml.MappingNode {
		return []ValidationIssue{joinIssue(IssueKindMissingRequired, name, "", "preset entry must be a mapping")}
	}
	var reposNode *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		if key != nil && key.Value == "repos" {
			reposNode = value
			break
		}
	}
	if reposNode == nil {
		return []ValidationIssue{joinIssue(IssueKindMissingRequired, name, "", "repos")}
	}
	if reposNode.Kind != yaml.SequenceNode {
		return []ValidationIssue{joinIssue(IssueKindMissingRequired, name, "", "repos must be a list")}
	}

	var issues []ValidationIssue
	var foundRepo bool
	for _, entry := range reposNode.Content {
		repoSpec, ok := repoFromNode(entry)
		if !ok {
			issues = append(issues, joinIssue(IssueKindInvalidRepoSpec, name, "", "repo entry must be a string or {repo: ...}"))
			continue
		}
		trimmed := strings.TrimSpace(repoSpec)
		if trimmed == "" {
			issues = append(issues, joinIssue(IssueKindInvalidRepoSpec, name, "", "repo spec is empty"))
			continue
		}
		foundRepo = true
		if _, _, err := repo.Normalize(trimmed); err != nil {
			issues = append(issues, joinIssue(IssueKindInvalidRepoSpec, name, trimmed, err.Error()))
		}
	}
	if !foundRepo && len(issues) == 0 {
		issues = append(issues, joinIssue(IssueKindMissingRequired, name, "", "repos"))
	}
	return issues
}

func repoFromNode(node *yaml.Node) (string, bool) {
	if node == nil {
		return "", false
	}
	if node.Kind == yaml.ScalarNode {
		return node.Value, true
	}
	if node.Kind != yaml.MappingNode {
		return "", false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		if key != nil && key.Value == "repo" && value != nil && value.Kind == yaml.ScalarNode {
			return value.Value, true
		}
	}
	return "", false
}

func joinFileIssue(err error) ValidationIssue {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return joinIssue(IssueKindFile, "", "", message)
}

func joinIssue(kind, presetName, repoSpec, message string) ValidationIssue {
	return ValidationIssue{
		Kind:    kind,
		Preset:  strings.TrimSpace(presetName),
		Repo:    strings.TrimSpace(repoSpec),
		Message: strings.TrimSpace(message),
	}
}
