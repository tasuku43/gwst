package apply

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tasuku43/gwst/internal/app/add"
	"github.com/tasuku43/gwst/internal/app/create"
	"github.com/tasuku43/gwst/internal/app/manifestplan"
	"github.com/tasuku43/gwst/internal/app/remove_repo"
	"github.com/tasuku43/gwst/internal/app/rm"
	"github.com/tasuku43/gwst/internal/domain/manifest"
	"github.com/tasuku43/gwst/internal/domain/repo"
	"github.com/tasuku43/gwst/internal/domain/workspace"
	"github.com/tasuku43/gwst/internal/infra/gitcmd"
)

type Options struct {
	AllowDirty       bool
	AllowStatusError bool
	PrefetchTimeout  time.Duration
	PrefetchOK       bool
	Step             func(text string)
}

func Apply(ctx context.Context, rootDir string, plan manifestplan.Result, opts Options) error {
	for _, change := range plan.Changes {
		if change.Kind != manifestplan.WorkspaceRemove {
			continue
		}
		logStep(opts.Step, fmt.Sprintf("remove workspace %s", change.WorkspaceID))
		if err := rm.Remove(ctx, rootDir, change.WorkspaceID, opts.AllowDirty); err != nil {
			return err
		}
	}

	for _, change := range plan.Changes {
		if change.Kind != manifestplan.WorkspaceUpdate {
			continue
		}
		if err := applyRepoRemovals(ctx, rootDir, change, opts); err != nil {
			return err
		}
		if err := applyRepoBranchRenames(ctx, rootDir, change, opts.Step); err != nil {
			return err
		}
	}

	for _, change := range plan.Changes {
		switch change.Kind {
		case manifestplan.WorkspaceAdd:
			if err := applyWorkspaceAdd(ctx, rootDir, plan.Desired, change, opts); err != nil {
				return err
			}
		case manifestplan.WorkspaceUpdate:
			if err := applyRepoAdds(ctx, rootDir, plan.Desired, change, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyWorkspaceAdd(ctx context.Context, rootDir string, desired manifest.File, change manifestplan.WorkspaceChange, opts Options) error {
	ws, ok := desired.Workspaces[change.WorkspaceID]
	if !ok {
		return fmt.Errorf("workspace not found in manifest: %s", change.WorkspaceID)
	}
	logStep(opts.Step, fmt.Sprintf("create workspace %s", change.WorkspaceID))
	_, err := create.CreateWorkspace(ctx, rootDir, change.WorkspaceID, workspace.Metadata{
		Description: ws.Description,
		Mode:        ws.Mode,
		PresetName:  ws.PresetName,
		SourceURL:   ws.SourceURL,
	})
	if err != nil {
		return err
	}
	baseBranchToRecord := ""
	baseBranchMixed := false
	fetch := true
	if opts.PrefetchOK {
		fetch = false
	}
	for _, repoEntry := range ws.Repos {
		logStep(opts.Step, fmt.Sprintf("worktree add %s", repoEntry.Alias))
		if strings.EqualFold(strings.TrimSpace(ws.Mode), workspace.MetadataModeReview) {
			if err := applyReviewRepoAdd(ctx, rootDir, change.WorkspaceID, repoEntry); err != nil {
				return err
			}
			continue
		}
		_, createdBranch, baseBranch, err := add.AddRepo(ctx, rootDir, change.WorkspaceID, repoEntry.RepoKey, repoEntry.Alias, repoEntry.Branch, repoEntry.BaseRef, fetch)
		if err != nil {
			return err
		}
		if createdBranch {
			baseBranchToRecord, baseBranchMixed = updateBaseBranchCandidate(baseBranchToRecord, baseBranchMixed, baseBranch)
		}
	}
	if baseBranchMixed {
		// Workspace-level base_branch can't represent multiple different bases across repos.
		// Keep it empty so `gion import` doesn't inject an incorrect base_ref into every repo.
		baseBranchToRecord = ""
	}
	if err := recordBaseBranchIfMissing(rootDir, change.WorkspaceID, baseBranchToRecord); err != nil {
		return err
	}
	return nil
}

func applyReviewRepoAdd(ctx context.Context, rootDir, workspaceID string, repoEntry manifest.Repo) error {
	repoSpec := repo.SpecFromKey(repoEntry.RepoKey)
	_, exists, err := repo.Exists(rootDir, repoSpec)
	if err != nil {
		return err
	}
	if !exists {
		if _, err := repo.Get(ctx, rootDir, repoSpec); err != nil {
			return err
		}
	}
	store, err := repo.Open(ctx, rootDir, repoSpec, false)
	if err != nil {
		return err
	}

	branch := strings.TrimSpace(repoEntry.Branch)
	if branch == "" {
		return fmt.Errorf("branch is required")
	}
	remoteRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
	if _, ok, err := gitcmd.ShowRef(ctx, store.StorePath, remoteRef); err != nil {
		return err
	} else if !ok {
		gitcmd.Logf("git fetch origin %s", branch)
		if _, err := gitcmd.Run(ctx, []string{"fetch", "origin", branch}, gitcmd.Options{Dir: store.StorePath}); err != nil {
			return err
		}
		if _, ok, err := gitcmd.ShowRef(ctx, store.StorePath, remoteRef); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("ref not found: %s", remoteRef)
		}
	}

	_, err = workspace.AddWithTrackingBranch(ctx, rootDir, workspaceID, repoSpec, repoEntry.Alias, branch, remoteRef, false)
	return err
}

func applyRepoRemovals(ctx context.Context, rootDir string, change manifestplan.WorkspaceChange, opts Options) error {
	for _, repoChange := range change.Repos {
		switch repoChange.Kind {
		case manifestplan.RepoRemove, manifestplan.RepoUpdate:
			if canRenameRepoBranchInPlace(repoChange) {
				continue
			}
			logStep(opts.Step, fmt.Sprintf("worktree remove %s", repoChange.Alias))
			if err := remove_repo.RemoveRepo(ctx, rootDir, change.WorkspaceID, repoChange.Alias, remove_repo.Options{
				AllowDirty:       opts.AllowDirty,
				AllowStatusError: opts.AllowStatusError,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyRepoBranchRenames(ctx context.Context, rootDir string, change manifestplan.WorkspaceChange, step func(text string)) error {
	for _, repoChange := range change.Repos {
		if !canRenameRepoBranchInPlace(repoChange) {
			continue
		}
		worktreePath := workspace.WorktreePath(rootDir, change.WorkspaceID, repoChange.Alias)

		currentBranch, err := gitcmd.RevParse(ctx, worktreePath, "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}
		if strings.TrimSpace(currentBranch) != strings.TrimSpace(repoChange.FromBranch) {
			return fmt.Errorf("cannot rename branch: repo %q is on %q, want %q", repoChange.Alias, currentBranch, repoChange.FromBranch)
		}

		logStep(step, fmt.Sprintf("branch rename %s", repoChange.Alias))
		if err := gitcmd.BranchMove(ctx, worktreePath, repoChange.FromBranch, repoChange.ToBranch); err != nil {
			return err
		}
	}
	return nil
}

func applyRepoAdds(ctx context.Context, rootDir string, desired manifest.File, change manifestplan.WorkspaceChange, opts Options) error {
	baseBranchToRecord := ""
	baseBranchMixed := false
	fetch := true
	if opts.PrefetchOK {
		fetch = false
	}
	for _, repoChange := range change.Repos {
		switch repoChange.Kind {
		case manifestplan.RepoAdd:
			logStep(opts.Step, fmt.Sprintf("worktree add %s", repoChange.Alias))
			baseRef := desiredBaseRef(desired, change.WorkspaceID, repoChange.Alias)
			_, createdBranch, baseBranch, err := add.AddRepo(ctx, rootDir, change.WorkspaceID, repoChange.ToRepo, repoChange.Alias, repoChange.ToBranch, baseRef, fetch)
			if err != nil {
				return err
			}
			if createdBranch {
				baseBranchToRecord, baseBranchMixed = updateBaseBranchCandidate(baseBranchToRecord, baseBranchMixed, baseBranch)
			}
		case manifestplan.RepoUpdate:
			if canRenameRepoBranchInPlace(repoChange) {
				continue
			}
			logStep(opts.Step, fmt.Sprintf("worktree add %s", repoChange.Alias))
			baseRef := desiredBaseRef(desired, change.WorkspaceID, repoChange.Alias)
			_, createdBranch, baseBranch, err := add.AddRepo(ctx, rootDir, change.WorkspaceID, repoChange.ToRepo, repoChange.Alias, repoChange.ToBranch, baseRef, fetch)
			if err != nil {
				return err
			}
			if createdBranch {
				baseBranchToRecord, baseBranchMixed = updateBaseBranchCandidate(baseBranchToRecord, baseBranchMixed, baseBranch)
			}
		}
	}
	if baseBranchMixed {
		baseBranchToRecord = ""
	}
	if err := recordBaseBranchIfMissing(rootDir, change.WorkspaceID, baseBranchToRecord); err != nil {
		return err
	}
	return nil
}

func updateBaseBranchCandidate(candidate string, mixed bool, baseBranch string) (string, bool) {
	if mixed {
		return candidate, mixed
	}
	baseBranch = strings.TrimSpace(baseBranch)
	if baseBranch == "" {
		return candidate, mixed
	}
	if candidate == "" {
		return baseBranch, mixed
	}
	if candidate != baseBranch {
		return candidate, true
	}
	return candidate, mixed
}

func canRenameRepoBranchInPlace(change manifestplan.RepoChange) bool {
	if change.Kind != manifestplan.RepoUpdate {
		return false
	}
	fromRepo := strings.TrimSpace(change.FromRepo)
	toRepo := strings.TrimSpace(change.ToRepo)
	fromBranch := strings.TrimSpace(change.FromBranch)
	toBranch := strings.TrimSpace(change.ToBranch)
	if fromRepo == "" || toRepo == "" || fromBranch == "" || toBranch == "" {
		return false
	}
	if fromRepo != toRepo {
		return false
	}
	if fromBranch == toBranch {
		return false
	}
	return true
}

func logStep(step func(text string), text string) {
	if step == nil {
		return
	}
	step(text)
}

func desiredBaseRef(desired manifest.File, workspaceID, alias string) string {
	ws, ok := desired.Workspaces[workspaceID]
	if !ok {
		return ""
	}
	for _, repoEntry := range ws.Repos {
		if strings.TrimSpace(repoEntry.Alias) == strings.TrimSpace(alias) {
			return strings.TrimSpace(repoEntry.BaseRef)
		}
	}
	return ""
}

func recordBaseBranchIfMissing(rootDir, workspaceID, baseBranch string) error {
	baseBranch = strings.TrimSpace(baseBranch)
	if baseBranch == "" {
		return nil
	}
	wsDir := workspace.WorkspaceDir(rootDir, workspaceID)
	meta, err := workspace.LoadMetadata(wsDir)
	if err != nil {
		return err
	}
	if strings.TrimSpace(meta.BaseBranch) != "" {
		return nil
	}
	meta.BaseBranch = baseBranch
	return workspace.SaveMetadata(wsDir, meta)
}
