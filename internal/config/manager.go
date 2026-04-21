package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	if err := m.SaveBaseConfig(BaseConfig{Version: CONFIG_VERSION, Skills: []Skill{}}); err != nil {
		return err
	}

	// Initialize Groups Config
	if err := m.SaveGroupsConfig(GroupsConfig{Version: CONFIG_VERSION, Groups: []Group{}}); err != nil {
		return err
	}

	// Initialize Summary Config with global context
	summary := SummaryConfig{Version: CONFIG_VERSION, Contexts: []ContextSummary{
		{
			Dir:         "global",
			ConfigPath:  filepath.Join(m.configDir, FILENAME_GLOBAL_CONTEXT),
			LastUpdated: "", // Can be filled with current time
		},
	}}
	if err := m.SaveSummaryConfig(summary); err != nil {
		return err
	}

	// Initialize Global Context Config
	globalContext := ContextConfig{
		Version:        CONFIG_VERSION,
		Scope:          "global",
		TargetDir:      m.configDir,
		ActiveSkills:   []SkillUsage{},
		InactiveSkills: []SkillUsage{},
		ActiveGroups:   []string{},
	}
	if err := m.SaveContextConfig(globalContext, FILENAME_GLOBAL_CONTEXT); err != nil {
		return err
	}

	return nil
}

func (m *Manager) SaveBaseConfig(cfg BaseConfig) error {
	return m.saveJSON(FILENAME_SKILLS, cfg)
}

func (m *Manager) LoadBaseConfig() (BaseConfig, error) {
	var cfg BaseConfig
	err := m.loadJSON(FILENAME_SKILLS, &cfg)
	if err != nil && os.IsNotExist(err) {
		return BaseConfig{Version: CONFIG_VERSION, Skills: []Skill{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveGroupsConfig(cfg GroupsConfig) error {
	return m.saveJSON(FILENAME_GROUPS, cfg)
}

func (m *Manager) LoadGroupsConfig() (GroupsConfig, error) {
	var cfg GroupsConfig
	err := m.loadJSON(FILENAME_GROUPS, &cfg)
	if err != nil && os.IsNotExist(err) {
		return GroupsConfig{Version: CONFIG_VERSION, Groups: []Group{}}, nil
	}
	return cfg, err
}

func (m *Manager) SaveSummaryConfig(cfg SummaryConfig) error {
	return m.saveJSON(FILENAME_SUMMARY, cfg)
}

func (m *Manager) LoadSummaryConfig() (SummaryConfig, error) {
	var cfg SummaryConfig
	err := m.loadJSON(FILENAME_SUMMARY, &cfg)
	if err != nil && os.IsNotExist(err) {
		return SummaryConfig{Version: CONFIG_VERSION, Contexts: []ContextSummary{}}, nil
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

// GetGroup finds a group by name.
func (m *Manager) GetGroup(name string) (*Group, error) {
	cfg, err := m.LoadGroupsConfig()
	if err != nil {
		return nil, err
	}
	for i, g := range cfg.Groups {
		if g.Name == name {
			return &cfg.Groups[i], nil
		}
	}
	return nil, nil
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
	cfg.Groups = append(cfg.Groups, Group{
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
	newGroups := []Group{}
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
				return nil // Not in group
			}
			cfg.Groups[i].SkillIDs = newSkillIDs
			return m.SaveGroupsConfig(cfg)
		}
	}
	return fmt.Errorf("group '%s' not found", groupName)
}

// SyncContext synchronizes the symlinks in the target directory based on the context config.
func (m *Manager) SyncContext(targetDir string, ctxCfg *ContextConfig, force bool) error {
	baseCfg, err := m.LoadBaseConfig()
	if err != nil {
		return err
	}
	groupsCfg, err := m.LoadGroupsConfig()
	if err != nil {
		return err
	}

	skillMap := make(map[string]Skill)
	for _, s := range baseCfg.Skills {
		skillMap[s.ID] = s
	}

	groupMap := make(map[string]Group)
	for _, g := range groupsCfg.Groups {
		groupMap[g.Name] = g
	}

	// Calculate desired state: skillID -> []agents
	desiredSkills := make(map[string][]string)

	// Add individual active skills
	for _, usage := range ctxCfg.ActiveSkills {
		desiredSkills[usage.ID] = mergeAgents(desiredSkills[usage.ID], usage.Agents)
	}

	// Add skills from active groups
	// Groups are applied to all existing agents in the project
	existingAgents := FindExistingAgents(targetDir)
	for _, groupName := range ctxCfg.ActiveGroups {
		if group, ok := groupMap[groupName]; ok {
			for _, skillID := range group.SkillIDs {
				desiredSkills[skillID] = mergeAgents(desiredSkills[skillID], existingAgents)
			}
		}
	}

	// For each agent, collect which skills should be there
	agentToSkills := make(map[string]map[string]string) // agent -> skillName -> skillPath
	for skillID, agents := range desiredSkills {
		skill, ok := skillMap[skillID]
		if !ok {
			continue
		}
		for _, agent := range agents {
			if agentToSkills[agent] == nil {
				agentToSkills[agent] = make(map[string]string)
			}
			agentToSkills[agent][skill.Name] = skill.Path
		}
	}

	// Calculate all agents that need to be processed
	allAgents := make(map[string]bool)
	for _, agent := range existingAgents {
		allAgents[agent] = true
	}
	for agent := range agentToSkills {
		allAgents[agent] = true
	}

	// Apply symlinks for all agents
	for agent := range allAgents {
		skills := agentToSkills[agent]
		if skills == nil {
			skills = make(map[string]string)
		}

		skillsDir, err := EnsureAgentSkillsDir(targetDir, agent)
		if err != nil {
			return err
		}

		// List current files in skills directory to find orphans
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			return err
		}
		currentFiles := make(map[string]bool)
		for _, entry := range entries {
			currentFiles[entry.Name()] = true
		}

		for skillName, skillPath := range skills {
			linkPath := filepath.Join(skillsDir, skillName)
			delete(currentFiles, skillName)

			if info, err := os.Lstat(linkPath); err == nil {
				// File exists, check if it's correct
				if info.Mode()&os.ModeSymlink != 0 {
					target, err := os.Readlink(linkPath)
					if err == nil && target == skillPath {
						// Already correct
						continue
					}
				}

				// Conflict or incorrect
				if !force {
					return fmt.Errorf("conflict: %s already exists and points elsewhere. use --force to overwrite", linkPath)
				}
				if err := os.Remove(linkPath); err != nil {
					return err
				}
			}

			if err := os.Symlink(skillPath, linkPath); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", linkPath, err)
			}
		}

		// Clean up orphans
		for orphan := range currentFiles {
			orphanPath := filepath.Join(skillsDir, orphan)
			info, err := os.Lstat(orphanPath)
			if err != nil {
				continue
			}
			// Only remove if it's a symlink pointing into our skill library (to be safe)
			// Actually, if it's in our managed skillsDir, we should probably just remove it if it's not in the list.
			// But the requirement just says "刪除組, 將組中的 skill 移除專案".
			// Let's only remove symlinks.
			if info.Mode()&os.ModeSymlink != 0 {
				os.Remove(orphanPath)
			}
		}
	}

	return nil
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

func mergeAgents(old, new []string) []string {
	m := make(map[string]bool)
	for _, a := range old {
		m[a] = true
	}
	for _, a := range new {
		m[a] = true
	}
	res := []string{}
	for a := range m {
		res = append(res, a)
	}
	return res
}

// AddSkillToContext adds a skill to the specified context config.
func (m *Manager) AddSkillToContext(cfg *ContextConfig, skillID string, agents []string) {
	// Remove from Inactive if it exists there
	newInactive := []SkillUsage{}
	for _, usage := range cfg.InactiveSkills {
		if usage.ID != skillID {
			newInactive = append(newInactive, usage)
		}
	}
	cfg.InactiveSkills = newInactive

	// Check if already in Active
	for i, usage := range cfg.ActiveSkills {
		if usage.ID == skillID {
			cfg.ActiveSkills[i].Agents = mergeAgents(usage.Agents, agents)
			return
		}
	}

	// Add to Active
	cfg.ActiveSkills = append(cfg.ActiveSkills, SkillUsage{ID: skillID, Agents: agents})
}

// EnsureLocalContext ensures a local context exists or creates a new one.
func EnsureLocalContext(targetDir string) (ContextConfig, error) {
	path := filepath.Join(targetDir, FILENAME_CONTEXT)
	var cfg ContextConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ContextConfig{
				Version:        CONFIG_VERSION,
				Scope:          "project",
				TargetDir:      targetDir,
				ActiveSkills:   []SkillUsage{},
				InactiveSkills: []SkillUsage{},
				ActiveGroups:   []string{},
			}, nil
		}
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

// SaveLocalContext saves the local context config.
func SaveLocalContext(targetDir string, cfg ContextConfig) error {
	path := filepath.Join(targetDir, FILENAME_CONTEXT)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// RemoveSkillFromContext completely removes a skill from the context.
func (m *Manager) RemoveSkillFromContext(cfg *ContextConfig, skillID string) {
	newActive := []SkillUsage{}
	for _, usage := range cfg.ActiveSkills {
		if usage.ID != skillID {
			newActive = append(newActive, usage)
		}
	}
	cfg.ActiveSkills = newActive

	newInactive := []SkillUsage{}
	for _, usage := range cfg.InactiveSkills {
		if usage.ID != skillID {
			newInactive = append(newInactive, usage)
		}
	}
	cfg.InactiveSkills = newInactive
}

// StopSkillInContext moves a skill from active to inactive.
func (m *Manager) StopSkillInContext(cfg *ContextConfig, skillID string, agents []string) {
	var targetUsage *SkillUsage
	newActive := []SkillUsage{}

	for _, usage := range cfg.ActiveSkills {
		if usage.ID == skillID {
			if len(agents) == 0 {
				// Stop all agents
				targetUsage = &usage
			} else {
				// Stop specific agents
				newAgents := []string{}
				agentMap := make(map[string]bool)
				for _, a := range agents {
					agentMap[a] = true
				}
				for _, a := range usage.Agents {
					if !agentMap[a] {
						newAgents = append(newAgents, a)
					}
				}

				if len(newAgents) > 0 {
					usage.Agents = newAgents
					newActive = append(newActive, usage)
				} else {
					// All agents for this skill are stopped
					targetUsage = &usage
				}
			}
		} else {
			newActive = append(newActive, usage)
		}
	}
	cfg.ActiveSkills = newActive

	if targetUsage != nil {
		// Move to InactiveSkills, merge if already there
		for i, usage := range cfg.InactiveSkills {
			if usage.ID == skillID {
				cfg.InactiveSkills[i].Agents = mergeAgents(usage.Agents, targetUsage.Agents)
				return
			}
		}
		cfg.InactiveSkills = append(cfg.InactiveSkills, *targetUsage)
	}
}
