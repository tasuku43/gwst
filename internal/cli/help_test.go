package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintCommandHelp_ManifestAliases(t *testing.T) {
	cases := []string{"manifest", "man", "m"}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			var buf bytes.Buffer
			if ok := printCommandHelp(cmd, &buf); !ok {
				t.Fatalf("expected ok=true")
			}
			out := buf.String()
			if !strings.Contains(out, "Usage: gwst manifest") {
				t.Fatalf("expected manifest usage, got:\n%s", out)
			}
		})
	}
}

func TestPrintCommandHelp_Ls_IsUnknown(t *testing.T) {
	var buf bytes.Buffer
	if ok := printCommandHelp("ls", &buf); ok {
		t.Fatalf("expected ok=false")
	}
}

func TestPrintCommandHelp_Preset_IsUnknown(t *testing.T) {
	var buf bytes.Buffer
	if ok := printCommandHelp("preset", &buf); ok {
		t.Fatalf("expected ok=false")
	}
}
