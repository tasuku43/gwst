package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListIncludesDescription(t *testing.T) {
	rootDir := t.TempDir()
	wsRoot := WorkspacesRoot(rootDir)
	if err := os.MkdirAll(wsRoot, 0o755); err != nil {
		t.Fatalf("create workspaces dir: %v", err)
	}

	ws1 := filepath.Join(wsRoot, "WS-1")
	if err := os.MkdirAll(ws1, 0o755); err != nil {
		t.Fatalf("create WS-1 dir: %v", err)
	}
	if err := SaveMetadata(ws1, Metadata{Description: "test description"}); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	ws2 := filepath.Join(wsRoot, "WS-2")
	if err := os.MkdirAll(ws2, 0o755); err != nil {
		t.Fatalf("create WS-2 dir: %v", err)
	}

	ws3 := filepath.Join(wsRoot, "WS-3")
	if err := os.MkdirAll(ws3, 0o755); err != nil {
		t.Fatalf("create WS-3 dir: %v", err)
	}
	badDir := filepath.Join(ws3, metadataDirName)
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatalf("create metadata dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(badDir, metadataFileName), []byte("{"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	entries, warnings, err := List(rootDir)
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected warnings for invalid metadata")
	}

	var ws1Desc string
	var ws2Desc string
	for _, entry := range entries {
		if entry.WorkspaceID == "WS-1" {
			ws1Desc = entry.Description
		}
		if entry.WorkspaceID == "WS-2" {
			ws2Desc = entry.Description
		}
	}
	if ws1Desc != "test description" {
		t.Fatalf("WS-1 description = %q, want %q", ws1Desc, "test description")
	}
	if ws2Desc != "" {
		t.Fatalf("WS-2 description = %q, want empty", ws2Desc)
	}
}
