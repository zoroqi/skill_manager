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

	var remainingSkills []internal.Skill
	if registryName != "" {
		for _, s := range cfg.Skills {
			if s.RegistryName != registryName {
				remainingSkills = append(remainingSkills, s)
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
			}
		}
	}

	cfg.Skills = append(remainingSkills, skills...)

	if err := a.store.SaveBaseConfig(cfg); err != nil {
		return 0, err
	}

	_ = a.cleanupSkills()

	return len(skills), nil
}

