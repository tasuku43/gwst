package workspace

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	MetadataDirName  = ".gion"
	metadataFileName = "metadata.json"
)

const (
	MetadataModePreset = "preset"
	MetadataModeRepo   = "repo"
	MetadataModeReview = "review"
	MetadataModeIssue  = "issue"
	MetadataModeResume = "resume"
	MetadataModeAdd    = "add"
)

type Metadata struct {
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode,omitempty"`
	PresetName  string `json:"preset_name,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	BaseBranch  string `json:"base_branch,omitempty"`
}

func LoadMetadata(wsDir string) (Metadata, error) {
	if strings.TrimSpace(wsDir) == "" {
		return Metadata{}, fmt.Errorf("workspace dir is required")
	}
	path := metadataPath(wsDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Metadata{}, nil
		}
		return Metadata{}, fmt.Errorf("read metadata: %w", err)
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, fmt.Errorf("parse metadata: %w", err)
	}
	return meta, nil
}

func SaveMetadata(wsDir string, meta Metadata) error {
	if strings.TrimSpace(wsDir) == "" {
		return fmt.Errorf("workspace dir is required")
	}
	meta = normalizeMetadata(meta)
	if meta == (Metadata{}) {
		return nil
	}
	if err := validateMetadata(meta); err != nil {
		return err
	}
	dir := filepath.Join(wsDir, MetadataDirName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create metadata dir: %w", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath(wsDir), data, 0o600); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func ReadDescription(wsDir string) (string, error) {
	meta, err := LoadMetadata(wsDir)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(meta.Description), nil
}

func metadataPath(wsDir string) string {
	return filepath.Join(wsDir, MetadataDirName, metadataFileName)
}

func normalizeMetadata(meta Metadata) Metadata {
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Mode = strings.TrimSpace(meta.Mode)
	meta.PresetName = strings.TrimSpace(meta.PresetName)
	meta.SourceURL = strings.TrimSpace(meta.SourceURL)
	meta.BaseBranch = strings.TrimSpace(meta.BaseBranch)
	return meta
}

func validateMetadata(meta Metadata) error {
	if meta.Mode != "" {
		switch meta.Mode {
		case MetadataModePreset, MetadataModeRepo, MetadataModeReview, MetadataModeIssue, MetadataModeResume, MetadataModeAdd:
		default:
			return fmt.Errorf("unsupported metadata mode: %s", meta.Mode)
		}
	}
	if meta.Mode == MetadataModePreset && meta.PresetName == "" {
		return fmt.Errorf("metadata preset_name is required for preset mode")
	}
	if meta.SourceURL != "" {
		parsed, err := url.ParseRequestURI(meta.SourceURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid metadata source_url: %s", meta.SourceURL)
		}
	}
	if meta.BaseBranch != "" {
		if strings.ContainsAny(meta.BaseBranch, " \t\r\n") {
			return fmt.Errorf("invalid metadata base_branch: %s", meta.BaseBranch)
		}
		if !strings.HasPrefix(meta.BaseBranch, "origin/") || meta.BaseBranch == "origin/" {
			return fmt.Errorf("invalid metadata base_branch (must be origin/<branch>): %s", meta.BaseBranch)
		}
	}
	return nil
}
