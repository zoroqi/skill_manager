package app

import (
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/scanner"
)

func (a *App) ScanSkills(dir string, registryName string) (int, error) {

	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return 0, err
	}

	skills, err := scanner.Scan(dir, registryName)
	if err != nil || len(skills) == 0 {
		return 0, err
	}

	skillMap := make(map[string]internal.Skill)
	for _, s := range cfg.Skills {
		skillMap[s.ID] = s
	}
	for _, s := range skills {
		skillMap[s.ID] = s
	}

	newSkills := make([]internal.Skill, 0, len(skillMap))
	for _, s := range skillMap {
		newSkills = append(newSkills, s)
	}
	cfg.Skills = newSkills

	if err := a.store.SaveBaseConfig(cfg); err != nil {
		return 0, err
	}
	return len(skills), nil
}
