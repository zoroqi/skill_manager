// internal/types.go
package internal

const (
	CONFIG_VERSION          = "1.0"
	FILENAME_SKILLS         = "skills.json"
	FILENAME_GROUPS         = "groups.json"
	FILENAME_SUMMARY        = "summary.json"
	FILENAME_REGISTRIES     = "registries.json"
	FILENAME_CONTEXT        = ".context.json"
	FILENAME_GLOBAL_CONTEXT = "global_context.json"
)

type Registry struct {
	Name        string `json:"name"`         // 註冊名稱 (如 user/repo)
	URL         string `json:"url"`          // 遠端 Git URL
	LocalPath   string `json:"local_path"`   // 本地存放目錄
	LastUpdated string `json:"last_updated"` // 最後更新時間
}

type RegistriesConfig struct {
	Version    string     `json:"version"`
	Registries []Registry `json:"registries"`
}

type Skill struct {
	ID           string         `json:"id" yaml:"-"`
	Name         string         `json:"name" yaml:"name"`
	RegistryName string         `json:"registry_name,omitempty" yaml:"-"` // 註冊名稱 (如果是從 register 安裝的)
	Path         string         `json:"path" yaml:"-"`
	Description  string         `json:"description" yaml:"description"`
	Metadata     map[string]any `json:"metadata" yaml:",inline"`
}

type BaseConfig struct {
	Version string  `json:"version"`
	Skills  []Skill `json:"skills"`
}

type Group struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SkillIDs    []string `json:"skill_ids"`
}

type GroupsConfig struct {
	Version string  `json:"version"`
	Groups  []Group `json:"groups"`
}

type SkillUsage struct {
	ID     string   `json:"id"`
	Agents []string `json:"agents"`
}

type ContextConfig struct {
	Version        string       `json:"version"`
	Scope          string       `json:"scope"`
	TargetDir      string       `json:"target_dir"`
	ActiveSkills   []SkillUsage `json:"active_skills"`
	InactiveSkills []SkillUsage `json:"inactive_skills,omitempty"`
	ActiveGroups   []string     `json:"active_groups"`
}

type ContextSummary struct {
	Dir         string `json:"dir"`
	ConfigPath  string `json:"config_path"`
	LastUpdated string `json:"last_updated"`
}

type SummaryConfig struct {
	Version  string           `json:"version"`
	Contexts []ContextSummary `json:"contexts"`
}
