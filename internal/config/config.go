// internal/config/config.go
package config

const (
	CONFIG_VERSION          = "1.0"
	FILENAME_SKILLS         = "skills.json"
	FILENAME_GROUPS         = "groups.json"
	FILENAME_SUMMARY        = "summary.json"
	FILENAME_CONTEXT        = ".context.json"
	FILENAME_GLOBAL_CONTEXT = "global_context.json"
)

type Skill struct {
	ID          string         `json:"id" yaml:"-"`
	Name        string         `json:"name" yaml:"name"`
	Path        string         `json:"path" yaml:"-"`
	Description string         `json:"description" yaml:"description"`
	Metadata    map[string]any `json:"metadata" yaml:",inline"`
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
