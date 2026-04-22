package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func (a *App) syncContext(targetDir string, ctxCfg *internal.ContextConfig, force bool) error {
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

	desiredSkills := make(map[string][]string)
	for _, usage := range ctxCfg.ActiveSkills {
		desiredSkills[usage.ID] = mergeAgents(desiredSkills[usage.ID], usage.Agents)
	}

	existingAgents := store.FindExistingAgents(targetDir)
	for _, groupName := range ctxCfg.ActiveGroups {
		group, ok := groupMap[groupName]
		if !ok {
			continue
		}
		for _, skillID := range group.SkillIDs {
			desiredSkills[skillID] = mergeAgents(desiredSkills[skillID], existingAgents)
		}
	}

	agentToSkills := make(map[string]map[string]string)
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

	allAgents := make(map[string]bool)
	for _, agent := range existingAgents {
		allAgents[agent] = true
	}
	for agent := range agentToSkills {
		allAgents[agent] = true
	}

	for agent := range allAgents {
		skills := agentToSkills[agent]
		if skills == nil {
			skills = make(map[string]string)
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

	return nil
}

func (a *App) SyncContext(workDir string, global, force bool) error {
	targetDir, ctxCfg, _, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}
	return a.syncContext(targetDir, ctxCfg, force)
}

func (a *App) GetStatus(workDir string, global bool) (internal.ContextConfig, []internal.Skill, error) {
	var ctxCfg internal.ContextConfig
	var err error

	baseCfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return ctxCfg, nil, fmt.Errorf("failed to load base config: %w", err)
	}

	if global {
		ctxCfg, err = a.store.LoadContextConfig(internal.FILENAME_GLOBAL_CONTEXT)
		if err != nil {
			return ctxCfg, nil, fmt.Errorf("failed to load global context: %w", err)
		}
	} else {
		targetDir := workDir
		if targetDir == "" {
			targetDir, err = os.Getwd()
			if err != nil {
				return ctxCfg, nil, err
			}
		}
		ctxCfg, err = store.EnsureLocalContext(targetDir)
		if err != nil {
			return ctxCfg, nil, fmt.Errorf("failed to load local context: %w", err)
		}
	}

	return ctxCfg, baseCfg.Skills, nil
}
