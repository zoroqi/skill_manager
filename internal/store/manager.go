// internal/store/manager.go
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/zoroqi/skill_manager/internal"
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
	if err := m.SaveBaseConfig(internal.BaseConfig{Version: internal.CONFIG_VERSION, Skills: []internal.Skill{}}); err != nil {
		return err
	}

	// Initialize Groups Config
	if err := m.SaveGroupsConfig(internal.GroupsConfig{Version: internal.CONFIG_VERSION, Groups: []internal.Group{}}); err != nil {
		return err
	}

	// Initialize Summary Config with global context
	summary := internal.SummaryConfig{Version: internal.CONFIG_VERSION, Contexts: []internal.ContextSummary{
		{
			Dir:         "global",
			ConfigPath:  filepath.Join(m.configDir, internal.FILENAME_GLOBAL_CONTEXT),
			LastUpdated: "", // Can be filled with current time
		},
	}}
	if err := m.SaveSummaryConfig(summary); err != nil {
		return err
	}

	// Initialize Global Context Config
	globalContext := internal.ContextConfig{
		Version:        internal.CONFIG_VERSION,
		Scope:          "global",
		TargetDir:      m.configDir,
		ActiveSkills:   []internal.SkillUsage{},
		InactiveSkills: []internal.SkillUsage{},
		ActiveGroups:   []string{},
	}
	if err := m.SaveContextConfig(globalContext, internal.FILENAME_GLOBAL_CONTEXT); err != nil {
		return err
	}

	return nil
}

func (m *Manager) SaveBaseConfig(cfg internal.BaseConfig) error {
	return m.saveJSON(internal.FILENAME_SKILLS, cfg)
}

func (m *Manager) LoadBaseConfig() (internal.BaseConfig, error) {
	var cfg internal.BaseConfig
	err := m.loadJSON(internal.FILENAME_SKILLS, &cfg)
	if err != nil && os.IsNotExist(err) {
		return internal.BaseConfig{Version: internal.CONFIG_VERSION, Skills: []internal.Skill{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveGroupsConfig(cfg internal.GroupsConfig) error {
	sort.Slice(cfg.Groups, func(i, j int) bool {
		return cfg.Groups[i].Name < cfg.Groups[j].Name
	})
	return m.saveJSON(internal.FILENAME_GROUPS, cfg)
}

func (m *Manager) LoadGroupsConfig() (internal.GroupsConfig, error) {
	var cfg internal.GroupsConfig
	err := m.loadJSON(internal.FILENAME_GROUPS, &cfg)
	if err != nil && os.IsNotExist(err) {
		return internal.GroupsConfig{Version: internal.CONFIG_VERSION, Groups: []internal.Group{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveSummaryConfig(cfg internal.SummaryConfig) error {
	return m.saveJSON(internal.FILENAME_SUMMARY, cfg)
}

func (m *Manager) LoadSummaryConfig() (internal.SummaryConfig, error) {
	var cfg internal.SummaryConfig
	err := m.loadJSON(internal.FILENAME_SUMMARY, &cfg)
	if err != nil && os.IsNotExist(err) {
		return internal.SummaryConfig{Version: internal.CONFIG_VERSION, Contexts: []internal.ContextSummary{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveContextConfig(cfg internal.ContextConfig, filename string) error {
	return m.saveJSON(filename, cfg)
}

func (m *Manager) LoadContextConfig(filename string) (internal.ContextConfig, error) {
	var cfg internal.ContextConfig
	err := m.loadJSON(filename, &cfg)
	return cfg, err
}

// CreateGroup adds a new empty group.
func (m *Manager) CreateGroup(name, description string) error {
	cfg, err := m.LoadGroupsConfig()
	if err != nil {
		return err
	}
	for _, g := range cfg.Groups {
		if g.Name == name {
			return fmt.Errorf("group '%s' already exists", name)
		}
	}
	cfg.Groups = append(cfg.Groups, internal.Group{
		ID:          name, // Using name as ID for simplicity
		Name:        name,
		Description: description,
		SkillIDs:    []string{},
	})
	return m.SaveGroupsConfig(cfg)
}

// DeleteGroup removes a group.
func (m *Manager) DeleteGroup(name string) error {
	cfg, err := m.LoadGroupsConfig()
	if err != nil {
		return err
	}
	newGroups := []internal.Group{}
	found := false
	for _, g := range cfg.Groups {
		if g.Name == name {
			found = true
			continue
		}
		newGroups = append(newGroups, g)
	}
	if !found {
		return fmt.Errorf("group '%s' not found", name)
	}
	cfg.Groups = newGroups
	return m.SaveGroupsConfig(cfg)
}

// AddSkillToGroup adds a skill ID to a group.
func (m *Manager) AddSkillToGroup(groupName, skillID string) error {
	cfg, err := m.LoadGroupsConfig()
	if err != nil {
		return err
	}
	for i, g := range cfg.Groups {
		if g.Name == groupName {
			if slices.Contains(g.SkillIDs, skillID) {
				return nil // Already exists
			}
			cfg.Groups[i].SkillIDs = append(cfg.Groups[i].SkillIDs, skillID)
			return m.SaveGroupsConfig(cfg)
		}
	}
	return fmt.Errorf("group '%s' not found", groupName)
}

// RemoveSkillFromGroup removes a skill ID from a group.
func (m *Manager) RemoveSkillFromGroup(groupName, skillID string) error {
	cfg, err := m.LoadGroupsConfig()
	if err != nil {
		return err
	}
	for i, g := range cfg.Groups {
		if g.Name == groupName {
			newSkillIDs := []string{}
			found := false
			for _, id := range g.SkillIDs {
				if id == skillID {
					found = true
					continue
				}
				newSkillIDs = append(newSkillIDs, id)
			}
			if !found {
				return fmt.Errorf("skill '%s' not found", skillID)
			}
			cfg.Groups[i].SkillIDs = newSkillIDs
			return m.SaveGroupsConfig(cfg)
		}
	}
	return fmt.Errorf("group '%s' not found", groupName)
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

// EnsureLocalContext ensures a local context exists or creates a new one.
func EnsureLocalContext(targetDir string) (internal.ContextConfig, error) {
	path := filepath.Join(targetDir, internal.FILENAME_CONTEXT)
	var cfg internal.ContextConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return internal.ContextConfig{
				Version:        internal.CONFIG_VERSION,
				Scope:          "project",
				TargetDir:      targetDir,
				ActiveSkills:   []internal.SkillUsage{},
				InactiveSkills: []internal.SkillUsage{},
				ActiveGroups:   []string{},
			}, nil
		}
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

// SaveLocalContext saves the local context config.
func SaveLocalContext(targetDir string, cfg internal.ContextConfig) error {
	path := filepath.Join(targetDir, internal.FILENAME_CONTEXT)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
