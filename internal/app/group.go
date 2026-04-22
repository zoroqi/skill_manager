package app

import (
	"fmt"

	"github.com/zoroqi/skill_manager/internal"
)

func (a *App) getGroup(name string) (*internal.Group, error) {
	cfg, err := a.store.LoadGroupsConfig()
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

func (a *App) CreateGroup(name, description string) error {
	return a.store.CreateGroup(name, description)
}

func (a *App) DeleteGroup(name string) error {
	return a.store.DeleteGroup(name)
}

func (a *App) AddSkillToGroup(groupName string, skillNames []string) error {
	baseCfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return err
	}
	for _, sn := range skillNames {
		var skillID string
		for _, s := range baseCfg.Skills {
			if s.Name == sn {
				skillID = s.ID
				break
			}
		}
		if skillID == "" {
			return fmt.Errorf("skill '%s' not found", sn)
		}
		if err := a.store.AddSkillToGroup(groupName, skillID); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) RemoveSkillFromGroup(groupName string, skillNames []string) error {
	baseCfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return err
	}
	for _, sn := range skillNames {
		var skillID string
		for _, s := range baseCfg.Skills {
			if s.Name == sn {
				skillID = s.ID
				break
			}
		}
		if skillID == "" {
			return fmt.Errorf("skill '%s' not found in base config, skipping", sn)
		}
		if err := a.store.RemoveSkillFromGroup(groupName, skillID); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) ListGroups() (internal.GroupsConfig, error) {
	return a.store.LoadGroupsConfig()
}
