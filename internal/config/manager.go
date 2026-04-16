package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Manager struct {
	configDir string
}

func NewManager(configDir string) *Manager {
	return &Manager{configDir: configDir}
}

// GetDefaultConfigDir returns the default configuration directory.
// It checks the SKILL_MANAGER_CONFIG environment variable first,
// and defaults to ~/.skill_manager if not set.
func GetDefaultConfigDir() (string, error) {
	if dir := os.Getenv("SKILL_MANAGER_CONFIG"); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".skill_manager"), nil
}

func (m *Manager) Init() error {
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return err
	}

	// Initialize Base Config
	if err := m.SaveBaseConfig(BaseConfig{Version: "1.0", Skills: []Skill{}}); err != nil {
		return err
	}

	// Initialize Groups Config
	if err := m.SaveGroupsConfig(GroupsConfig{Version: "1.0", Groups: []Group{}}); err != nil {
		return err
	}

	// Initialize Summary Config with global context
	summary := SummaryConfig{Version: "1.0", Contexts: []ContextSummary{
		{
			Dir:         "global",
			ConfigPath:  filepath.Join(m.configDir, "global_context.json"),
			LastUpdated: "", // Can be filled with current time
		},
	}}
	if err := m.SaveSummaryConfig(summary); err != nil {
		return err
	}

	// Initialize Global Context Config
	globalContext := ContextConfig{
		Version:      "1.0",
		Scope:        "global",
		TargetDir:    m.configDir,
		ActiveSkills: []string{},
		ActiveGroups: []string{},
	}
	if err := m.SaveContextConfig(globalContext, "global_context.json"); err != nil {
		return err
	}

	return nil
}

func (m *Manager) SaveBaseConfig(cfg BaseConfig) error {
	return m.saveJSON("skills.json", cfg)
}

func (m *Manager) LoadBaseConfig() (BaseConfig, error) {
	var cfg BaseConfig
	err := m.loadJSON("skills.json", &cfg)
	if err != nil && os.IsNotExist(err) {
		return BaseConfig{Version: "1.0", Skills: []Skill{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveGroupsConfig(cfg GroupsConfig) error {
	return m.saveJSON("groups.json", cfg)
}

func (m *Manager) LoadGroupsConfig() (GroupsConfig, error) {
	var cfg GroupsConfig
	err := m.loadJSON("groups.json", &cfg)
	if err != nil && os.IsNotExist(err) {
		return GroupsConfig{Version: "1.0", Groups: []Group{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveSummaryConfig(cfg SummaryConfig) error {
	return m.saveJSON("summary.json", cfg)
}

func (m *Manager) LoadSummaryConfig() (SummaryConfig, error) {
	var cfg SummaryConfig
	err := m.loadJSON("summary.json", &cfg)
	if err != nil && os.IsNotExist(err) {
		return SummaryConfig{Version: "1.0", Contexts: []ContextSummary{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveContextConfig(cfg ContextConfig, filename string) error {
	return m.saveJSON(filename, cfg)
}

func (m *Manager) LoadContextConfig(filename string) (ContextConfig, error) {
	var cfg ContextConfig
	err := m.loadJSON(filename, &cfg)
	return cfg, err
}

func (m *Manager) saveJSON(filename string, data any) error {
	path := filepath.Join(m.configDir, filename)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

func (m *Manager) loadJSON(filename string, v any) error {
	path := filepath.Join(m.configDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
