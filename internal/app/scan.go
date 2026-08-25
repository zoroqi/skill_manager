package app

import (
	"path/filepath"
	"strings"

	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/scanner"
)

func (a *App) ScanSkills(dir string, registryName string) (int, error) {
	cfg, err := a.store.LoadBaseConfig()
	if err != nil {
		return 0, err
	}

	skills, err := scanner.Scan(dir, registryName)
	if err != nil {
		return 0, err
	}

	scannedIDs := make(map[string]bool, len(skills))
	for _, s := range skills {
		scannedIDs[s.ID] = true
	}

	var remainingSkills []internal.Skill
	var newlyRemoved []internal.Skill
	if registryName != "" {
		for _, s := range cfg.Skills {
			if s.RegistryName != registryName {
				remainingSkills = append(remainingSkills, s)
			} else if !scannedIDs[s.ID] {
				newlyRemoved = append(newlyRemoved, s)
			}
		}
	} else {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			absDir = dir
		}
		for _, s := range cfg.Skills {
			if s.RegistryName != "" || !strings.HasPrefix(s.Path, absDir) {
				remainingSkills = append(remainingSkills, s)
			} else if !scannedIDs[s.ID] {
				newlyRemoved = append(newlyRemoved, s)
			}
		}
	}

	deletedMap := make(map[string]internal.Skill)
	for _, d := range cfg.DeletedSkills {
		deletedMap[d.ID] = d
	}
	for _, r := range newlyRemoved {
		deletedMap[r.ID] = r
	}
	for _, s := range skills {
		delete(deletedMap, s.ID)
	}

	newDeleted := make([]internal.Skill, 0, len(deletedMap))
	for _, d := range deletedMap {
		newDeleted = append(newDeleted, d)
	}
	cfg.DeletedSkills = newDeleted
	cfg.Skills = append(remainingSkills, skills...)

	if err := a.store.SaveBaseConfig(cfg); err != nil {
		return 0, err
	}

	_ = a.cleanupSkills()

	return len(skills), nil
}

