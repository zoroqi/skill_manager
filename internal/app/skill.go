package app

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

func addSkillToContext(cfg *internal.ContextConfig, skillID string, agents []string) {
	newInactive := make([]internal.SkillUsage, 0, len(cfg.InactiveSkills))
	for _, usage := range cfg.InactiveSkills {
		if usage.ID != skillID {
			newInactive = append(newInactive, usage)
		}
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

func removeSkillFromContext(cfg *internal.ContextConfig, skillID string) {
	newActive := make([]internal.SkillUsage, 0, len(cfg.ActiveSkills))
	for _, usage := range cfg.ActiveSkills {
		if usage.ID != skillID {
			newActive = append(newActive, usage)
		}
	}
	cfg.ActiveSkills = newActive

	newInactive := make([]internal.SkillUsage, 0, len(cfg.InactiveSkills))
	for _, usage := range cfg.InactiveSkills {
		if usage.ID != skillID {
			newInactive = append(newInactive, usage)
		}
	}
	cfg.InactiveSkills = newInactive
}

func stopSkillInContext(cfg *internal.ContextConfig, skillID string, agents []string) {
	var targetUsage *internal.SkillUsage
	newActive := make([]internal.SkillUsage, 0, len(cfg.ActiveSkills))

	for _, usage := range cfg.ActiveSkills {
		if usage.ID != skillID {
			newActive = append(newActive, usage)
			continue
		}

		if len(agents) == 0 {
			targetUsage = &usage
			continue
		}

		newAgents := make([]string, 0, len(usage.Agents))
		agentMap := make(map[string]bool, len(agents))
		for _, agent := range agents {
			agentMap[agent] = true
		}
		for _, agent := range usage.Agents {
			if !agentMap[agent] {
				newAgents = append(newAgents, agent)
			}
		}

		if len(newAgents) > 0 {
			usage.Agents = newAgents
			newActive = append(newActive, usage)
			continue
		}

		targetUsage = &usage
	}
	cfg.ActiveSkills = newActive

	if targetUsage == nil {
		return
	}

	for i, usage := range cfg.InactiveSkills {
		if usage.ID == skillID {
			cfg.InactiveSkills[i].Agents = mergeAgents(usage.Agents, targetUsage.Agents)
			return
		}
	}

	cfg.InactiveSkills = append(cfg.InactiveSkills, *targetUsage)
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

	addSkillToContext(ctxCfg, targetSkill.ID, agents)

	if err := a.syncContext(targetDir, ctxCfg, force); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) AddGroup(workDir, groupName string, global, force bool) error {
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

	if !slices.Contains(ctxCfg.ActiveGroups, groupName) {
		ctxCfg.ActiveGroups = append(ctxCfg.ActiveGroups, groupName)
	}

	if err := a.syncContext(targetDir, ctxCfg, force); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) RemoveSkill(workDir, skillName string, global bool) error {
	targetDir, ctxCfg, saveCtx, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}

	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return fmt.Errorf("failed to load base config: %w", err)
	}

	var skillID string
	for _, s := range cfg.Skills {
		if s.Name == skillName {
			skillID = s.ID
			break
		}
	}
	if skillID == "" {
		return fmt.Errorf("skill '%s' not found in scanned library", skillName)
	}

	removeSkillFromContext(ctxCfg, skillID)

	if err := a.syncContext(targetDir, ctxCfg, true); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) RemoveGroup(workDir, groupName string, global bool) error {
	targetDir, ctxCfg, saveCtx, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}

	newGroups := []string{}
	found := false
	for _, g := range ctxCfg.ActiveGroups {
		if g == groupName {
			found = true
			continue
		}
		newGroups = append(newGroups, g)
	}
	if !found {
		return fmt.Errorf("group '%s' not active in this context", groupName)
	}
	ctxCfg.ActiveGroups = newGroups

	if err := a.syncContext(targetDir, ctxCfg, true); err != nil {
		return err
	}
	return saveCtx()
}

func (a *App) StopSkill(workDir, skillName string, agents []string, global bool) error {
	targetDir, ctxCfg, saveCtx, err := a.resolveContext(workDir, global)
	if err != nil {
		return err
	}

	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return fmt.Errorf("failed to load base config: %w", err)
	}

	var skillID string
	for _, s := range cfg.Skills {
		if s.Name == skillName {
			skillID = s.ID
			break
		}
	}
	if skillID == "" {
		return fmt.Errorf("skill '%s' not found in scanned library", skillName)
	}

	if err := store.RemoveAgentSymlinks(targetDir, skillName, agents); err != nil {
		return err
	}

	stopSkillInContext(ctxCfg, skillID, agents)
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
			return a.store.SaveContextConfig(ctxCfg, internal.FILENAME_GLOBAL_CONTEXT)
		}
		return targetDir, &ctxCfg, saveCtx, nil
	}

	targetDir = workDir
	ctxCfg, err = store.EnsureLocalContext(targetDir)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to load local context: %w", err)
	}
	saveCtx := func() error {
		return store.SaveLocalContext(targetDir, ctxCfg)
	}
	return targetDir, &ctxCfg, saveCtx, nil
}
