package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/tasuku43/gwst/internal/core/gitcmd"
	"github.com/tasuku43/gwst/internal/core/paths"
	"github.com/tasuku43/gwst/internal/domain/repo"
)

func Add(ctx context.Context, rootDir, workspaceID, repoSpec, alias string, fetch bool) (Repo, error) {
	return AddWithBranch(ctx, rootDir, workspaceID, repoSpec, alias, workspaceID, "", fetch)
}

func AddWithBranch(ctx context.Context, rootDir, workspaceID, repoSpec, alias, branch, baseRef string, fetch bool) (Repo, error) {
	if err := validateBranchName(ctx, branch); err != nil {
		return Repo{}, err
	}
	prep, err := prepareAdd(ctx, rootDir, workspaceID, repoSpec, alias, fetch)
	if err != nil {
		return Repo{}, err
	}

	if baseRef == "" {
		baseRef, err = resolveBaseRef(ctx, prep.store.StorePath)
		if err != nil {
			return Repo{}, err
		}
	}

	branchExists, err := branchExistsInStore(ctx, prep.store.StorePath, branch)
	if err != nil {
		return Repo{}, err
	}

	if branchExists {
		gitcmd.Logf("git worktree add %s %s", prep.worktreePath, branch)
		if err := gitcmd.WorktreeAddExistingBranch(ctx, prep.store.StorePath, prep.worktreePath, branch); err != nil {
			return Repo{}, err
		}
	} else {
		gitcmd.Logf("git worktree add -b %s %s %s", branch, prep.worktreePath, baseRef)
		if err := gitcmd.WorktreeAddNewBranch(ctx, prep.store.StorePath, branch, prep.worktreePath, baseRef); err != nil {
			return Repo{}, err
		}
	}

	repoEntry := Repo{
		Alias:        prep.alias,
		RepoSpec:     repoSpec,
		RepoKey:      prep.spec.RepoKey,
		StorePath:    prep.store.StorePath,
		WorktreePath: prep.worktreePath,
		Branch:       branch,
	}

	return repoEntry, nil
}

func AddWithTrackingBranch(ctx context.Context, rootDir, workspaceID, repoSpec, alias, branch, remoteRef string, fetch bool) (Repo, error) {
	if err := validateBranchName(ctx, branch); err != nil {
		return Repo{}, err
	}
	if strings.TrimSpace(remoteRef) == "" {
		return Repo{}, fmt.Errorf("remote ref is required")
	}
	prep, err := prepareAdd(ctx, rootDir, workspaceID, repoSpec, alias, fetch)
	if err != nil {
		return Repo{}, err
	}

	remoteName := strings.TrimPrefix(remoteRef, "refs/remotes/")
	if remoteName == remoteRef {
		return Repo{}, fmt.Errorf("remote ref is required: %s", remoteRef)
	}
	exists, err := refExists(ctx, prep.store.StorePath, remoteRef)
	if err != nil {
		return Repo{}, err
	}
	if !exists {
		return Repo{}, fmt.Errorf("ref not found: %s", remoteRef)
	}

	gitcmd.Logf("git worktree add -b %s --track %s %s", branch, prep.worktreePath, remoteName)
	if err := gitcmd.WorktreeAddTrackingBranch(ctx, prep.store.StorePath, branch, prep.worktreePath, remoteName); err != nil {
		return Repo{}, err
	}

	repoEntry := Repo{
		Alias:        prep.alias,
		RepoSpec:     repoSpec,
		RepoKey:      prep.spec.RepoKey,
		StorePath:    prep.store.StorePath,
		WorktreePath: prep.worktreePath,
		Branch:       branch,
	}

	return repoEntry, nil
}

type addPrep struct {
	spec         repo.Spec
	alias        string
	store        repo.Store
	worktreePath string
}

func prepareAdd(ctx context.Context, rootDir, workspaceID, repoSpec, alias string, fetch bool) (addPrep, error) {
	if err := validateWorkspaceID(ctx, workspaceID); err != nil {
		return addPrep{}, err
	}
	if rootDir == "" {
		return addPrep{}, fmt.Errorf("root directory is required")
	}

	wsDir := WorkspaceDir(rootDir, workspaceID)
	if exists, err := paths.DirExists(wsDir); err != nil {
		return addPrep{}, err
	} else if !exists {
		return addPrep{}, fmt.Errorf("workspace does not exist: %s", wsDir)
	}

	spec, _, err := repo.Normalize(repoSpec)
	if err != nil {
		return addPrep{}, err
	}
	if alias == "" {
		alias = spec.Repo
	}
	if alias == "" {
		return addPrep{}, fmt.Errorf("alias is required")
	}
	repos, _, err := ScanRepos(ctx, wsDir)
	if err != nil {
		return addPrep{}, err
	}
	for _, existing := range repos {
		if existing.Alias == alias {
			return addPrep{}, fmt.Errorf("alias already exists: %s", alias)
		}
		if spec.RepoKey != "" && existing.RepoKey == spec.RepoKey {
			return addPrep{}, fmt.Errorf("repo already exists: %s", spec.RepoKey)
		}
	}

	store, err := repo.Open(ctx, rootDir, repoSpec, fetch)
	if err != nil {
		return addPrep{}, err
	}

	worktreePath := WorktreePath(rootDir, workspaceID, alias)
	if exists, err := paths.DirExists(worktreePath); err != nil {
		return addPrep{}, err
	} else if exists {
		return addPrep{}, fmt.Errorf("worktree already exists: %s", worktreePath)
	}

	return addPrep{
		spec:         spec,
		alias:        alias,
		store:        store,
		worktreePath: worktreePath,
	}, nil
}

func resolveBaseRef(ctx context.Context, storePath string) (string, error) {
	if storePath == "" {
		return "", fmt.Errorf("store path is required")
	}

	remoteHead, remoteErr := detectDefaultRemoteRef(ctx, storePath)
	if remoteErr == nil && remoteHead != "" {
		return remoteHead, nil
	}

	localHead, localErr := detectLocalHeadRef(ctx, storePath)
	if localErr == nil && localHead != "" {
		return localHead, nil
	}

	for _, candidate := range []string{"main", "master", "develop"} {
		exists, err := localRefExists(ctx, storePath, candidate)
		if err != nil {
			return "", err
		}
		if exists {
			return fmt.Sprintf("refs/heads/%s", candidate), nil
		}
	}

	for _, candidate := range []string{"origin/main", "origin/master", "origin/develop"} {
		exists, err := remoteRefExists(ctx, storePath, candidate)
		if err != nil {
			return "", err
		}
		if exists {
			return candidate, nil
		}
	}

	if remoteErr != nil {
		return "", remoteErr
	}
	if localErr != nil {
		return "", localErr
	}
	return "", fmt.Errorf("cannot detect default base ref")
}

func detectLocalHeadRef(ctx context.Context, storePath string) (string, error) {
	ref, ok, err := gitcmd.SymbolicRef(ctx, storePath, "HEAD")
	if err != nil {
		return "", err
	}
	if !ok || ref == "" {
		return "", nil
	}
	return ref, nil
}

func detectDefaultRemoteRef(ctx context.Context, storePath string) (string, error) {
	ref, ok, err := gitcmd.SymbolicRef(ctx, storePath, "refs/remotes/origin/HEAD")
	if err != nil {
		return "", err
	}
	if !ok || !strings.HasPrefix(ref, "refs/remotes/") {
		return "", nil
	}
	return strings.TrimPrefix(ref, "refs/remotes/"), nil
}

func localRefExists(ctx context.Context, storePath, name string) (bool, error) {
	fullRef := fmt.Sprintf("refs/heads/%s", name)
	return refExists(ctx, storePath, fullRef)
}

func remoteRefExists(ctx context.Context, storePath, ref string) (bool, error) {
	fullRef := fmt.Sprintf("refs/remotes/%s", ref)
	return refExists(ctx, storePath, fullRef)
}

func refExists(ctx context.Context, storePath, fullRef string) (bool, error) {
	_, exists, err := gitcmd.ShowRef(ctx, storePath, fullRef)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func branchExistsInStore(ctx context.Context, storePath, branch string) (bool, error) {
	ref := fmt.Sprintf("refs/heads/%s", branch)
	_, exists, err := gitcmd.ShowRef(ctx, storePath, ref)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func validateBranchName(ctx context.Context, branch string) error {
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("branch is required")
	}
	if err := gitcmd.CheckRefFormatBranch(ctx, branch); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}
	return nil
}

// ValidateBranchName checks whether the given branch name satisfies git's
// ref format rules. It mirrors the internal validation used by workspace
// operations so callers outside this package can pre-validate user input.
func ValidateBranchName(ctx context.Context, branch string) error {
	return validateBranchName(ctx, branch)
}
