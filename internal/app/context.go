package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func (a *App) syncContext(targetDir string, ctxCfg *internal.ContextConfig, force, global bool) error {
	baseCfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return err
	}

	groupsCfg, err := a.store.LoadGroupsConfig()
	if err != nil {
		return err
	}

	skillMap := make(map[string]internal.Skill, len(baseCfg.Skills))
	for _, skill := range baseCfg.Skills {
		skillMap[skill.ID] = skill
	}

	groupMap := make(map[string]internal.Group, len(groupsCfg.Groups))
	for _, group := range groupsCfg.Groups {
		groupMap[group.Name] = group
	}

	existingAgents := store.FindExistingAgents(targetDir)

	// 1. Process InactiveSkills for lookup
	inactive := make(map[string]map[string]bool) // skillName -> agent -> true
	for _, usage := range ctxCfg.InactiveSkills {
		if inactive[usage.Name] == nil {
			inactive[usage.Name] = make(map[string]bool)
		}
		if len(usage.Agents) == 0 {
			inactive[usage.Name][""] = true // "" means all agents
		} else {
			for _, agent := range usage.Agents {
				inactive[usage.Name][agent] = true
			}
		}
	}

	isInactive := func(skillName, agent string) bool {
		m := inactive[skillName]
		if m == nil {
			return false
		}
		return m[""] || m[agent]
	}

	// 2. Resolve final skills per agent
	// agent -> skillName -> skillPath
	agentToSkillPaths := make(map[string]map[string]string)

	// User-added skills (ActiveSkills) have highest priority
	for _, usage := range ctxCfg.ActiveSkills {
		skill, ok := skillMap[usage.ID]
		if !ok {
			continue
		}
		agents := usage.Agents
		if len(agents) == 0 {
			agents = existingAgents
		}
		for _, agent := range agents {
			if agentToSkillPaths[agent] == nil {
				agentToSkillPaths[agent] = make(map[string]string)
			}
			agentToSkillPaths[agent][skill.Name] = skill.Path
		}
	}

	// Process ActiveGroups with priority (alphabetical order)
	sortedGroups := make([]internal.GroupUsage, len(ctxCfg.ActiveGroups))
	copy(sortedGroups, ctxCfg.ActiveGroups)
	sort.Slice(sortedGroups, func(i, j int) bool {
		return sortedGroups[i].Name < sortedGroups[j].Name
	})

	for _, usage := range sortedGroups {
		group, ok := groupMap[usage.Name]
		if !ok {
			continue
		}
		agents := usage.Agents
		if len(agents) == 0 {
			agents = existingAgents
		}

		for _, skillID := range group.SkillIDs {
			skill, ok := skillMap[skillID]
			if !ok {
				continue
			}

			for _, agent := range agents {
				if isInactive(skill.Name, agent) {
					continue
				}

				if agentToSkillPaths[agent] == nil {
					agentToSkillPaths[agent] = make(map[string]string)
				}

				// Only add if not already set by higher priority source
				if _, exists := agentToSkillPaths[agent][skill.Name]; !exists {
					agentToSkillPaths[agent][skill.Name] = skill.Path
				}
			}
		}
	}

	allAgents := make(map[string]bool)
	for _, agent := range existingAgents {
		allAgents[agent] = true
	}
	for agent := range agentToSkillPaths {
		allAgents[agent] = true
	}

	var activeAgentDirs []string
	for agent := range allAgents {
		skills := agentToSkillPaths[agent]
		if skills == nil {
			skills = make(map[string]string)
		}

		if len(skills) > 0 {
			activeAgentDirs = append(activeAgentDirs, agent)
		}

		skillsDir, err := store.EnsureAgentSkillsDir(targetDir, agent)
		if err != nil {
			return err
		}

		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			return err
		}

		currentFiles := make(map[string]bool, len(entries))
		for _, entry := range entries {
			currentFiles[entry.Name()] = true
		}

		for skillName, skillPath := range skills {
			linkPath := filepath.Join(skillsDir, skillName)
			delete(currentFiles, skillName)

			if info, err := os.Lstat(linkPath); err == nil {
				if info.Mode()&os.ModeSymlink != 0 {
					target, err := os.Readlink(linkPath)
					if err == nil && target == skillPath {
						continue
					}
				}

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

		for orphan := range currentFiles {
			orphanPath := filepath.Join(skillsDir, orphan)
			info, err := os.Lstat(orphanPath)
			if err != nil {
				continue
			}

			if info.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(orphanPath); err != nil {
					return fmt.Errorf("failed to remove orphan symlink %s: %w", orphanPath, err)
				}
			}
		}
	}

	// Update .gitignore if not global
	if !global {
		var ignorePaths []string
		for agent, skills := range agentToSkillPaths {
			agentDir := store.GetAgentDir(agent)
			for skillName := range skills {
				ignorePaths = append(ignorePaths, fmt.Sprintf("%s/skills/%s", agentDir, skillName))
			}
		}
		sort.Strings(ignorePaths)
		if err := updateGitignore(targetDir, ignorePaths); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}

	return nil
}

func (a *App) SyncContext(workDir string, global, force bool) error {
	targetDir, ctxCfg, _, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}
	return a.syncContext(targetDir, ctxCfg, force, global)
}

func (a *App) GetStatus(workDir string, global bool) (internal.ContextConfig, []internal.Skill, []internal.Group, error) {
	var ctxCfg internal.ContextConfig
	var err error

	baseCfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return ctxCfg, nil, nil, fmt.Errorf("failed to load base config: %w", err)
	}

	if global {
		ctxCfg, err = a.store.LoadContextConfig(internal.FILENAME_GLOBAL_CONTEXT)
		if err != nil {
			return ctxCfg, nil, nil, fmt.Errorf("failed to load global context: %w", err)
		}
	} else {
		targetDir := workDir
		if targetDir == "" {
			targetDir, err = os.Getwd()
			if err != nil {
				return ctxCfg, nil, nil, err
			}
		}
		ctxCfg, err = store.EnsureLocalContext(targetDir)
		if err != nil {
			return ctxCfg, nil, nil, fmt.Errorf("failed to load local context: %w", err)
		}
	}
	groups, err := a.store.LoadGroupsConfig()
	if err != nil {
		return ctxCfg, nil, nil, fmt.Errorf("failed to load groups: %w", err)
	}
	return ctxCfg, baseCfg.Skills, groups.Groups, nil
}
