// internal/config/config.go
package config

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

type ContextConfig struct {
	Version      string   `json:"version"`
	Scope        string   `json:"scope"`
	TargetDir    string   `json:"target_dir"`
	ActiveSkills []string `json:"active_skills"`
	ActiveGroups []string `json:"active_groups"`
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
