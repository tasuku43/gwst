package manifest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"gopkg.in/yaml.v3"
)

type ValidationIssue struct {
	// Ref is a logical reference path like:
	// - gwst.yaml
	// - workspaces
	// - workspaces.PROJ-123.repos[0].branch
	Ref     string
	Message string
}

type ValidationResult struct {
	Path   string
	Issues []ValidationIssue
}

type ValidationError struct {
	Result ValidationResult
}

func (e *ValidationError) Error() string {
	return "manifest validation failed"
}

var presetNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func Validate(ctx context.Context, rootDir string) (ValidationResult, error) {
	if strings.TrimSpace(rootDir) == "" {
		return ValidationResult{}, fmt.Errorf("root directory is required")
	}
	path := filepath.Join(rootDir, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return ValidationResult{
			Path:   path,
			Issues: []ValidationIssue{{Ref: "gwst.yaml", Message: err.Error()}},
		}, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return ValidationResult{
			Path:   path,
			Issues: []ValidationIssue{{Ref: "gwst.yaml", Message: fmt.Sprintf("invalid yaml (%s)", strings.TrimSpace(err.Error()))}},
		}, nil
	}

	root := unwrapDocument(&doc)
	var issues []ValidationIssue
	issues = append(issues, validateVersion(root)...)
	issues = append(issues, validateWorkspaces(ctx, root)...)
	issues = append(issues, validatePresets(root)...)
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

func validateVersion(root *yaml.Node) []ValidationIssue {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	versionNode := mappingValue(root, "version")
	if versionNode == nil {
		return nil
	}
	if versionNode.Kind != yaml.ScalarNode {
		return []ValidationIssue{{Ref: "version", Message: "invalid value (must be an integer)"}}
	}
	v, err := strconv.Atoi(strings.TrimSpace(versionNode.Value))
	if err != nil {
		return []ValidationIssue{{Ref: "version", Message: "invalid value (must be an integer)"}}
	}
	if v != 1 {
		return []ValidationIssue{{Ref: "version", Message: fmt.Sprintf("unsupported version: %d (supported: 1)", v)}}
	}
	return nil
}

func validateWorkspaces(ctx context.Context, root *yaml.Node) []ValidationIssue {
	if root == nil || root.Kind != yaml.MappingNode {
		return []ValidationIssue{{Ref: "workspaces", Message: "missing required field"}}
	}
	workspacesNode := mappingValue(root, "workspaces")
	if workspacesNode == nil {
		return []ValidationIssue{{Ref: "workspaces", Message: "missing required field"}}
	}
	if workspacesNode.Kind != yaml.MappingNode {
		return []ValidationIssue{{Ref: "workspaces", Message: "invalid value (must be a mapping)"}}
	}

	var issues []ValidationIssue
	seenIDs := map[string]struct{}{}

	for i := 0; i+1 < len(workspacesNode.Content); i += 2 {
		keyNode := workspacesNode.Content[i]
		valueNode := workspacesNode.Content[i+1]

		workspaceID := strings.TrimSpace(nodeStringValue(keyNode))
		if workspaceID == "" {
			issues = append(issues, ValidationIssue{Ref: "workspaces", Message: "workspace id is empty"})
			continue
		}
		if _, ok := seenIDs[workspaceID]; ok {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("workspaces.%s", workspaceID), Message: "duplicate workspace id"})
		} else {
			seenIDs[workspaceID] = struct{}{}
		}
		if err := workspace.ValidateWorkspaceID(ctx, workspaceID); err != nil {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("workspaces.%s", workspaceID), Message: err.Error()})
		}

		if valueNode == nil || valueNode.Kind != yaml.MappingNode {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("workspaces.%s", workspaceID), Message: "invalid value (workspace entry must be a mapping)"})
			continue
		}

		issues = append(issues, validateWorkspaceEntry(ctx, workspaceID, valueNode, root)...)
	}
	return issues
}

func validateWorkspaceEntry(ctx context.Context, workspaceID string, node *yaml.Node, root *yaml.Node) []ValidationIssue {
	var issues []ValidationIssue

	mode := strings.TrimSpace(scalarValue(mappingValue(node, "mode")))
	if mode != "" {
		switch mode {
		case workspace.MetadataModePreset, workspace.MetadataModeRepo, workspace.MetadataModeReview, workspace.MetadataModeIssue, workspace.MetadataModeResume, workspace.MetadataModeAdd:
		default:
			issues = append(issues, ValidationIssue{
				Ref:     fmt.Sprintf("workspaces.%s.mode", workspaceID),
				Message: fmt.Sprintf("invalid value: %s", mode),
			})
		}
	}

	presetName := strings.TrimSpace(scalarValue(mappingValue(node, "preset_name")))
	if mode == workspace.MetadataModePreset && presetName == "" {
		issues = append(issues, ValidationIssue{
			Ref:     fmt.Sprintf("workspaces.%s.preset_name", workspaceID),
			Message: "missing required field for preset mode",
		})
	}

	if presetName != "" {
		if presetsNode := mappingValue(root, "presets"); presetsNode != nil && presetsNode.Kind == yaml.MappingNode {
			if !mappingHasKey(presetsNode, presetName) {
				issues = append(issues, ValidationIssue{
					Ref:     fmt.Sprintf("workspaces.%s.preset_name", workspaceID),
					Message: fmt.Sprintf("preset not found: %s", presetName),
				})
			}
		}
	}

	sourceURL := strings.TrimSpace(scalarValue(mappingValue(node, "source_url")))
	if sourceURL != "" {
		parsed, err := url.ParseRequestURI(sourceURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			issues = append(issues, ValidationIssue{
				Ref:     fmt.Sprintf("workspaces.%s.source_url", workspaceID),
				Message: fmt.Sprintf("invalid url: %s", sourceURL),
			})
		}
	}

	reposNode := mappingValue(node, "repos")
	if reposNode == nil {
		issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("workspaces.%s.repos", workspaceID), Message: "missing required field"})
		return issues
	}
	if reposNode.Kind != yaml.SequenceNode {
		issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("workspaces.%s.repos", workspaceID), Message: "invalid value (must be a list)"})
		return issues
	}

	seenAliases := map[string]struct{}{}
	for i, entry := range reposNode.Content {
		refPrefix := fmt.Sprintf("workspaces.%s.repos[%d]", workspaceID, i)
		if entry == nil || entry.Kind != yaml.MappingNode {
			issues = append(issues, ValidationIssue{Ref: refPrefix, Message: "invalid value (repo entry must be a mapping)"})
			continue
		}

		alias := strings.TrimSpace(scalarValue(mappingValue(entry, "alias")))
		if alias == "" {
			issues = append(issues, ValidationIssue{Ref: refPrefix + ".alias", Message: "missing required field"})
		} else {
			if alias == ".gwst" {
				issues = append(issues, ValidationIssue{Ref: refPrefix + ".alias", Message: "invalid value: .gwst is reserved"})
			}
			if strings.Contains(alias, "/") || strings.Contains(alias, "\\") {
				issues = append(issues, ValidationIssue{Ref: refPrefix + ".alias", Message: "invalid value (must not contain path separators)"})
			}
			if _, ok := seenAliases[alias]; ok {
				issues = append(issues, ValidationIssue{Ref: refPrefix + ".alias", Message: fmt.Sprintf("duplicate alias %q", alias)})
			} else {
				seenAliases[alias] = struct{}{}
			}
		}

		repoKey := strings.TrimSpace(scalarValue(mappingValue(entry, "repo_key")))
		if repoKey == "" {
			issues = append(issues, ValidationIssue{Ref: refPrefix + ".repo_key", Message: "missing required field"})
		} else if err := validateRepoKey(repoKey); err != nil {
			issues = append(issues, ValidationIssue{Ref: refPrefix + ".repo_key", Message: err.Error()})
		}

		branch := strings.TrimSpace(scalarValue(mappingValue(entry, "branch")))
		if branch == "" {
			issues = append(issues, ValidationIssue{Ref: refPrefix + ".branch", Message: "missing required field"})
		} else if err := workspace.ValidateBranchName(ctx, branch); err != nil {
			issues = append(issues, ValidationIssue{Ref: refPrefix + ".branch", Message: err.Error()})
		}

		baseRef := strings.TrimSpace(scalarValue(mappingValue(entry, "base_ref")))
		if baseRef != "" {
			if !strings.HasPrefix(baseRef, "origin/") || baseRef == "origin/" {
				issues = append(issues, ValidationIssue{Ref: refPrefix + ".base_ref", Message: "invalid value (must be origin/<branch>)"})
			} else if err := workspace.ValidateBranchName(ctx, strings.TrimPrefix(baseRef, "origin/")); err != nil {
				issues = append(issues, ValidationIssue{Ref: refPrefix + ".base_ref", Message: fmt.Sprintf("invalid base ref: %v", err)})
			}
		}
	}

	return issues
}

func validatePresets(root *yaml.Node) []ValidationIssue {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	presetsNode := mappingValue(root, "presets")
	if presetsNode == nil {
		return nil
	}
	if presetsNode.Kind != yaml.MappingNode {
		return []ValidationIssue{{Ref: "presets", Message: "invalid value (must be a mapping)"}}
	}

	var issues []ValidationIssue
	seen := map[string]struct{}{}
	for i := 0; i+1 < len(presetsNode.Content); i += 2 {
		key := presetsNode.Content[i]
		value := presetsNode.Content[i+1]
		name := strings.TrimSpace(nodeStringValue(key))
		if name == "" {
			issues = append(issues, ValidationIssue{Ref: "presets", Message: "preset name is empty"})
			continue
		}
		if err := validatePresetName(name); err != nil {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("presets.%s", name), Message: err.Error()})
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("presets.%s", name), Message: "duplicate preset name"})
		} else {
			seen[name] = struct{}{}
		}
		issues = append(issues, validatePresetEntry(name, value)...)
	}
	return issues
}

func validatePresetName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("preset name is required")
	}
	if !presetNamePattern.MatchString(trimmed) {
		return fmt.Errorf("invalid preset name: %s", name)
	}
	return nil
}

func validatePresetEntry(name string, node *yaml.Node) []ValidationIssue {
	refPrefix := fmt.Sprintf("presets.%s", name)
	if node == nil || node.Kind != yaml.MappingNode {
		return []ValidationIssue{{Ref: refPrefix, Message: "invalid value (preset entry must be a mapping)"}}
	}
	reposNode := mappingValue(node, "repos")
	if reposNode == nil {
		return []ValidationIssue{{Ref: refPrefix + ".repos", Message: "missing or empty"}}
	}
	if reposNode.Kind != yaml.SequenceNode {
		return []ValidationIssue{{Ref: refPrefix + ".repos", Message: "invalid value (must be a list)"}}
	}

	var issues []ValidationIssue
	var foundRepo bool
	for i, entry := range reposNode.Content {
		repoSpec, ok := presetRepoFromNode(entry)
		if !ok {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("%s.repos[%d]", refPrefix, i), Message: "invalid value (must be a string or {repo: ...})"})
			continue
		}
		trimmed := strings.TrimSpace(repoSpec)
		if trimmed == "" {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("%s.repos[%d]", refPrefix, i), Message: "repo spec is empty"})
			continue
		}
		foundRepo = true
		if _, _, err := repo.Normalize(trimmed); err != nil {
			issues = append(issues, ValidationIssue{Ref: fmt.Sprintf("%s.repos[%d]", refPrefix, i), Message: err.Error()})
		}
	}
	if !foundRepo && len(issues) == 0 {
		issues = append(issues, ValidationIssue{Ref: refPrefix + ".repos", Message: "missing or empty"})
	}
	return issues
}

func presetRepoFromNode(node *yaml.Node) (string, bool) {
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

func validateRepoKey(repoKey string) error {
	if strings.ContainsAny(repoKey, " \t\r\n") {
		return fmt.Errorf("invalid repo key (must not contain whitespace)")
	}
	trimmed := strings.TrimSuffix(strings.TrimSpace(repoKey), ".git")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 {
		return fmt.Errorf("invalid repo key (must be host/owner/repo[.git])")
	}
	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[2]) == "" {
		return fmt.Errorf("invalid repo key (must be host/owner/repo[.git])")
	}
	return nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i]
		v := node.Content[i+1]
		if k != nil && k.Value == key {
			return v
		}
	}
	return nil
}

func mappingHasKey(node *yaml.Node, key string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i]
		if k != nil && strings.TrimSpace(k.Value) == strings.TrimSpace(key) {
			return true
		}
	}
	return false
}

func scalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}

func nodeStringValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return node.Value
}
