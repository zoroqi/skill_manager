package app

import (
	"path/filepath"
	"sort"

	"github.com/zoroqi/skill_manager/internal"
)

// GetSummaryReport gathers the configured skills and groups for each project recorded in summary.json.
func (a *App) GetSummaryReport() (*internal.SummaryReport, error) {
	summaryCfg, err := a.store.LoadSummaryConfig()
	if err != nil {
		return nil, err
	}

	items := make([]internal.ContextSummaryItem, 0, len(summaryCfg.Contexts))

	for _, ctxSummary := range summaryCfg.Contexts {
		global := ctxSummary.Dir == "global"
		ctxCfg, skills, _, err := a.GetStatus(ctxSummary.Dir, global)
		if err != nil {
			// Skip context if failed to load
			continue
		}

		projectName := ctxCfg.Scope
		if global {
			if projectName == "" {
				projectName = "global"
			}
		} else {
			if projectName == "" {
				projectName = filepath.Base(ctxSummary.Dir)
			}
		}

		skillMap := make(map[string]internal.Skill, len(skills))
		for _, s := range skills {
			skillMap[s.ID] = s
		}

		// Extract and deduplicate skill names
		skillNameSet := make(map[string]bool)
		for _, su := range ctxCfg.ActiveSkills {
			name := su.ID
			if s, ok := skillMap[su.ID]; ok && s.Name != "" {
				name = s.Name
			}
			if name != "" {
				skillNameSet[name] = true
			}
		}
		skillList := make([]string, 0, len(skillNameSet))
		for name := range skillNameSet {
			skillList = append(skillList, name)
		}
		sort.Strings(skillList)

		// Extract and deduplicate group names
		groupNameSet := make(map[string]bool)
		for _, gu := range ctxCfg.ActiveGroups {
			if gu.Name != "" {
				groupNameSet[gu.Name] = true
			}
		}
		groupList := make([]string, 0, len(groupNameSet))
		for name := range groupNameSet {
			groupList = append(groupList, name)
		}
		sort.Strings(groupList)

		items = append(items, internal.ContextSummaryItem{
			ProjectName: projectName,
			Dir:         ctxSummary.Dir,
			Skills:      skillList,
			Groups:      groupList,
		})
	}

	return &internal.SummaryReport{
		Items: items,
	}, nil
}
