package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	metadataDirName  = ".gwst"
	metadataFileName = "metadata.json"
)

type Metadata struct {
	Description string `json:"description,omitempty"`
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
	meta.Description = strings.TrimSpace(meta.Description)
	if meta.Description == "" {
		return nil
	}
	dir := filepath.Join(wsDir, metadataDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create metadata dir: %w", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath(wsDir), data, 0o644); err != nil {
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
	return filepath.Join(wsDir, metadataDirName, metadataFileName)
}
