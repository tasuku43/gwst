package workspace

type Manifest struct {
	SchemaVersion int    `yaml:"schema_version"`
	WorkspaceID   string `yaml:"workspace_id"`
	CreatedAt     string `yaml:"created_at"`
	LastUsedAt    string `yaml:"last_used_at"`
	Policy        Policy `yaml:"policy"`
	Repos         []Repo `yaml:"repos"`
}

type Policy struct {
	Pinned  bool `yaml:"pinned"`
	TTLDays int  `yaml:"ttl_days"`
}

type Repo struct {
	Alias         string `yaml:"alias"`
	RepoSpec      string `yaml:"repo_spec"`
	RepoKey       string `yaml:"repo_key"`
	StorePath     string `yaml:"store_path"`
	WorktreePath  string `yaml:"worktree_path"`
	Branch        string `yaml:"branch"`
	BaseRef       string `yaml:"base_ref"`
	CreatedBranch bool   `yaml:"created_branch"`
}
