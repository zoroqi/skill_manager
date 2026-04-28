package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func addSkillToContext(cfg *internal.ContextConfig, skillID, skillName string, agents []string) {
	newInactive := make([]internal.InactiveSkillUsage, 0, len(cfg.InactiveSkills))
	for _, usage := range cfg.InactiveSkills {
		if usage.Name != skillName {
			newInactive = append(newInactive, usage)
		} else if len(agents) > 0 && len(usage.Agents) > 0 {
			// Partial unmasking
			remainingMasked := []string{}
			toUnmask := make(map[string]bool)
			for _, a := range agents {
				toUnmask[a] = true
			}
			for _, a := range usage.Agents {
				if !toUnmask[a] {
					remainingMasked = append(remainingMasked, a)
				}
			}
			if len(remainingMasked) > 0 {
				usage.Agents = remainingMasked
				newInactive = append(newInactive, usage)
			}
			continue
		} else if len(agents) == 0 {
			// If adding without specific agents, unmask entirely?
			// Or maybe adding a skill should always unmask it for those agents.
			continue
		}
		// If adding for specific agents but usage.Agents is nil (all),
		// we might need to handle it, but for simplicity let's just unmask.
	}
	cfg.InactiveSkills = newInactive

	for i, usage := range cfg.ActiveSkills {
		if usage.ID == skillID {
			cfg.ActiveSkills[i].Agents = mergeAgents(usage.Agents, agents)
			return
		}
	}

	cfg.ActiveSkills = append(cfg.ActiveSkills, internal.SkillUsage{ID: skillID, Agents: agents})
}

func removeSkillFromContext(cfg *internal.ContextConfig, skillID string, agents []string) {
	newActive := make([]internal.SkillUsage, 0, len(cfg.ActiveSkills))
	for _, usage := range cfg.ActiveSkills {
		if usage.ID == skillID {
			if len(agents) == 0 {
				continue // Remove entirely
			}
			// Partial removal
			remainingAgents := []string{}
			toRemove := make(map[string]bool)
			for _, a := range agents {
				toRemove[a] = true
			}
			for _, a := range usage.Agents {
				if !toRemove[a] {
					remainingAgents = append(remainingAgents, a)
				}
			}
			if len(remainingAgents) > 0 {
				usage.Agents = remainingAgents
				newActive = append(newActive, usage)
			}
			continue
		}
		newActive = append(newActive, usage)
	}
	cfg.ActiveSkills = newActive
}

func (a *App) resolveSkill(skillName, registryName string) (*internal.Skill, error) {
	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	var targetSkills []internal.Skill
	for _, s := range cfg.Skills {
		if s.Name == skillName {
			if registryName == "" || s.RegistryName == registryName {
				targetSkills = append(targetSkills, s)
			}
		}
	}

	if len(targetSkills) == 0 {
		return nil, fmt.Errorf("skill '%s' not found. please scan it first", skillName)
	}

	if len(targetSkills) > 1 {
		var registries []string
		for _, s := range targetSkills {
			reg := s.RegistryName
			if reg == "" {
				reg = "local"
			}
			registries = append(registries, reg)
		}
		return nil, fmt.Errorf("multiple skills found with name '%s' in registries: %s. please specify the registry using --register flag", skillName, strings.Join(registries, ", "))
	}

	return &targetSkills[0], nil
}

func (a *App) ListSkills() ([]internal.Skill, error) {
	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Skills, nil
}

func (a *App) AddSkill(workDir, skillName, registryName string, agents []string, global, force bool) error {
	targetDir, ctxCfg, saveCtx, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}

	targetSkill, err := a.resolveSkill(skillName, registryName)
	if err != nil {
		return err
	}

	if len(agents) == 0 {
		agents = store.FindExistingAgents(targetDir)
		if len(agents) == 0 {
			return fmt.Errorf("no agent directories found. Please specify --agent flag (e.g. --agent gemini) to create one")
		}
	}

	addSkillToContext(ctxCfg, targetSkill.ID, targetSkill.Name, agents)

	if err := a.syncContext(targetDir, ctxCfg, force, global); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) AddGroup(workDir, groupName string, agents []string, global, force bool) error {
	targetDir, ctxCfg, saveCtx, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}

	group, err := a.getGroup(groupName)
	if err != nil {
		return err
	}
	if group == nil {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	found := false
	for i, usage := range ctxCfg.ActiveGroups {
		if usage.Name == groupName {
			ctxCfg.ActiveGroups[i].Agents = mergeAgents(usage.Agents, agents)
			found = true
			break
		}
	}
	if !found {
		ctxCfg.ActiveGroups = append(ctxCfg.ActiveGroups, internal.GroupUsage{Name: groupName, Agents: agents})
	}

	if err := a.syncContext(targetDir, ctxCfg, force, global); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) RemoveSkill(workDir, skillName string, agents []string, global bool) error {
	targetDir, ctxCfg, saveCtx, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}

	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return fmt.Errorf("failed to load base config: %w", err)
	}

	groupsCfg, err := a.store.LoadGroupsConfig()
	if err != nil {
		return fmt.Errorf("failed to load groups config: %w", err)
	}

	var matchingIDs []string
	for _, s := range cfg.Skills {
		if s.Name == skillName {
			matchingIDs = append(matchingIDs, s.ID)
		}
	}
	if len(matchingIDs) == 0 {
		return fmt.Errorf("skill '%s' not found in scanned library", skillName)
	}

	groupMap := make(map[string]internal.Group)
	for _, g := range groupsCfg.Groups {
		groupMap[g.Name] = g
	}

	isSkillInActiveGroups := false
	for _, activeGroup := range ctxCfg.ActiveGroups {
		g, ok := groupMap[activeGroup.Name]
		if !ok {
			continue
		}
		for _, id := range g.SkillIDs {
			// Find if any matching skill (by name) is in this group
			if slices.Contains(matchingIDs, id) {
				isSkillInActiveGroups = true
			}
			if isSkillInActiveGroups {
				break
			}
		}
		if isSkillInActiveGroups {
			break
		}
	}

	for _, skillID := range matchingIDs {
		removeSkillFromContext(ctxCfg, skillID, agents)
	}

	if isSkillInActiveGroups {
		found := false
		for i, usage := range ctxCfg.InactiveSkills {
			if usage.Name == skillName {
				if len(agents) == 0 {
					ctxCfg.InactiveSkills[i].Agents = nil
				} else if ctxCfg.InactiveSkills[i].Agents != nil {
					ctxCfg.InactiveSkills[i].Agents = mergeAgents(ctxCfg.InactiveSkills[i].Agents, agents)
				}
				found = true
				break
			}
		}
		if !found {
			ctxCfg.InactiveSkills = append(ctxCfg.InactiveSkills, internal.InactiveSkillUsage{Name: skillName, Agents: agents})
		}
	}

	if err := a.syncContext(targetDir, ctxCfg, true, global); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) RemoveGroup(workDir, groupName string, agents []string, global bool) error {
	targetDir, ctxCfg, saveCtx, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}

	newGroups := []internal.GroupUsage{}
	found := false
	for _, g := range ctxCfg.ActiveGroups {
		if g.Name == groupName {
			found = true
			if len(agents) == 0 {
				continue
			}
			// Handle partial agent removal
			currentAgents := g.Agents
			if len(currentAgents) == 0 {
				currentAgents = store.FindExistingAgents(targetDir)
			}

			agentMap := make(map[string]bool)
			for _, a := range agents {
				agentMap[a] = true
			}

			updatedAgents := []string{}
			for _, a := range currentAgents {
				if !agentMap[a] {
					updatedAgents = append(updatedAgents, a)
				}
			}

			if len(updatedAgents) > 0 {
				g.Agents = updatedAgents
				newGroups = append(newGroups, g)
			}
			continue
		}
		newGroups = append(newGroups, g)
	}
	if !found {
		return fmt.Errorf("group '%s' not active in this context", groupName)
	}
	ctxCfg.ActiveGroups = newGroups

	if err := a.syncContext(targetDir, ctxCfg, true, global); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) resolveContext(workDir string, global bool) (string, *internal.ContextConfig, func() error, error) {
	var targetDir string
	var ctxCfg internal.ContextConfig
	var err error

	if global {
		targetDir, err = os.UserHomeDir()
		if err != nil {
			return "", nil, nil, err
		}
		ctxCfg, err = a.store.LoadContextConfig(internal.FILENAME_GLOBAL_CONTEXT)
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to load global context: %w", err)
		}
		saveCtx := func() error {
			err := a.store.SaveContextConfig(ctxCfg, internal.FILENAME_GLOBAL_CONTEXT)
			if err == nil {
				_ = a.store.UpdateContextSummary("global", filepath.Join(a.store.GetConfigDir(), internal.FILENAME_GLOBAL_CONTEXT))
			}
			return err
		}
		return targetDir, &ctxCfg, saveCtx, nil
	}

	targetDir = workDir
	ctxCfg, err = store.EnsureLocalContext(targetDir)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to load local context: %w", err)
	}
	saveCtx := func() error {
		err := store.SaveLocalContext(targetDir, ctxCfg)
		if err == nil {
			_ = a.store.UpdateContextSummary(targetDir, filepath.Join(targetDir, internal.FILENAME_CONTEXT))
		}
		return err
	}
	return targetDir, &ctxCfg, saveCtx, nil
}
