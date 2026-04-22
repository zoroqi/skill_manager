package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zoroqi/skill_manager/internal"
)

func (m *Manager) GetGroup(name string) (*internal.Group, error) {
	cfg, err := m.LoadGroupsConfig()
	if err != nil {
		return nil, err
	}
	for i, group := range cfg.Groups {
		if group.Name == name {
			return &cfg.Groups[i], nil
		}
	}
	return nil, nil
}

func (m *Manager) SyncContext(targetDir string, ctxCfg *internal.ContextConfig, force bool) error {
	baseCfg, err := m.LoadBaseConfig()
	if err != nil {
		return err
	}

	groupsCfg, err := m.LoadGroupsConfig()
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
		desiredSkills[usage.ID] = mergeAgentsCompat(desiredSkills[usage.ID], usage.Agents)
	}

	existingAgents := FindExistingAgents(targetDir)
	for _, groupName := range ctxCfg.ActiveGroups {
		group, ok := groupMap[groupName]
		if !ok {
			continue
		}
		for _, skillID := range group.SkillIDs {
			desiredSkills[skillID] = mergeAgentsCompat(desiredSkills[skillID], existingAgents)
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

		skillsDir, err := EnsureAgentSkillsDir(targetDir, agent)
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

func mergeAgentsCompat(old, new []string) []string {
	m := make(map[string]bool, len(old)+len(new))
	for _, agent := range old {
		m[agent] = true
	}
	for _, agent := range new {
		m[agent] = true
	}
	res := make([]string, 0, len(m))
	for agent := range m {
		res = append(res, agent)
	}
	return res
}
