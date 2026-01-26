package cli

import (
	"bytes"
	"testing"

	"github.com/tasuku43/gion/internal/app/manifestplan"
	"github.com/tasuku43/gion/internal/domain/workspace"
	"github.com/tasuku43/gion/internal/infra/output"
	"github.com/tasuku43/gion/internal/ui"
)

func TestRenderWorkspaceRiskDetails_IndentsUnderBullet(t *testing.T) {
	var b bytes.Buffer
	renderer := ui.NewRenderer(&b, ui.DefaultTheme(), false)

	renderer.Section("Plan")
	renderer.BulletError("- remove workspace SREP-3810")

	status := workspace.StatusResult{
		WorkspaceID: "SREP-3810",
		Repos: []workspace.RepoStatus{
			{
				Alias:       "terraforms",
				Branch:      "SREP-3810-add-jobmaching-outsourcing",
				Upstream:    "origin/SREP-3810-add-jobmaching-outsourcing",
				AheadCount:  0,
				BehindCount: 0,
			},
		},
	}

	renderWorkspaceRiskDetails(renderer, status, output.Indent)

	want := "" +
		"Plan\n" +
		"  • - remove workspace SREP-3810\n" +
		"    └─ terraforms (branch: SREP-3810-add-jobmaching-outsourcing)\n" +
		"       sync: upstream=origin/SREP-3810-add-jobmaching-outsourcing ahead=0 behind=0\n"

	if got := b.String(); got != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderPlanWorkspaceAddRepos_IndentsUnderWorkspaceBullet(t *testing.T) {
	var b bytes.Buffer
	renderer := ui.NewRenderer(&b, ui.DefaultTheme(), false)

	renderer.Section("Plan")
	renderer.BulletSuccess("+ add workspace PROJ-123")

	renderPlanWorkspaceAddRepos(renderer, []manifestplan.RepoChange{
		{
			Kind:     manifestplan.RepoAdd,
			Alias:    "api",
			ToRepo:   "github.com/org/api",
			ToBranch: "PROJ-123",
		},
	})

	want := "" +
		"Plan\n" +
		"  • + add workspace PROJ-123\n" +
		"    └─ api (branch: PROJ-123)\n" +
		"       repo: github.com/org/api\n"

	if got := b.String(); got != want {
		t.Fatalf("unexpected output:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
