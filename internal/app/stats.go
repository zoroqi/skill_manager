package app

import (
	"sort"

	"github.com/zoroqi/skill_manager/internal"
)

// GetStatsReport gathers and computes usage statistics for all used skills and groups across registered contexts.
func (a *App) GetStatsReport() (*internal.StatsReport, error) {
	summaryCfg, err := a.store.LoadSummaryConfig()
	if err != nil {
		return nil, err
	}

	skillCountMap := make(map[string]int)
	groupCountMap := make(map[string]int)

	for _, ctxSummary := range summaryCfg.Contexts {
		global := ctxSummary.Dir == "global"
		ctxCfg, skills, groups, err := a.GetStatus(ctxSummary.Dir, global)
		if err != nil {
			// Skip context if failed to load
			continue
		}

		skillMap := make(map[string]internal.Skill, len(skills))
		for _, s := range skills {
			skillMap[s.ID] = s
		}

		groupMap := make(map[string]internal.Group, len(groups))
		for _, g := range groups {
			groupMap[g.Name] = g
		}

		// 1. Group usage counting
		ctxGroupNames := make(map[string]bool)
		for _, gu := range ctxCfg.ActiveGroups {
			if gu.Name != "" {
				ctxGroupNames[gu.Name] = true
			}
		}
		for gName := range ctxGroupNames {
			groupCountMap[gName]++
		}

		// 2. Inactive skills lookup
		inactive := make(map[string]map[string]bool)
		for _, usage := range ctxCfg.InactiveSkills {
			if inactive[usage.Name] == nil {
				inactive[usage.Name] = make(map[string]bool)
			}
			if len(usage.Agents) == 0 {
				inactive[usage.Name][""] = true
			} else {
				for _, ag := range usage.Agents {
					inactive[usage.Name][ag] = true
				}
			}
		}
		isSkillInactiveForAgent := func(skillName, agent string) bool {
			m := inactive[skillName]
			if m == nil {
				return false
			}
			return m[""] || m[agent]
		}

		// 3. Effective skills counting in this context
		ctxSkillNames := make(map[string]bool)

		// 3.1 Directly added active skills (ActiveSkills)
		for _, su := range ctxCfg.ActiveSkills {
			skill, ok := skillMap[su.ID]
			if !ok {
				continue
			}
			ctxSkillNames[skill.Name] = true
		}

		// 3.2 Skills expanded from active groups (ActiveGroups)
		for _, gu := range ctxCfg.ActiveGroups {
			group, ok := groupMap[gu.Name]
			if !ok {
				continue
			}
			for _, sid := range group.SkillIDs {
				skill, ok := skillMap[sid]
				if !ok {
					continue
				}

				if len(gu.Agents) == 0 {
					if !isSkillInactiveForAgent(skill.Name, "") {
						ctxSkillNames[skill.Name] = true
					}
				} else {
					hasActiveAgent := false
					for _, ag := range gu.Agents {
						if !isSkillInactiveForAgent(skill.Name, ag) {
							hasActiveAgent = true
							break
						}
					}
					if hasActiveAgent {
						ctxSkillNames[skill.Name] = true
					}
				}
			}
		}

		for sName := range ctxSkillNames {
			skillCountMap[sName]++
		}
	}

	// Build sorted Skill stats: only include used skills (Count > 0), Count desc, Name asc
	skillStats := make([]internal.StatItem, 0, len(skillCountMap))
	for name, count := range skillCountMap {
		if count > 0 {
			skillStats = append(skillStats, internal.StatItem{
				Name:  name,
				Count: count,
			})
		}
	}
	sort.Slice(skillStats, func(i, j int) bool {
		if skillStats[i].Count == skillStats[j].Count {
			return skillStats[i].Name < skillStats[j].Name
		}
		return skillStats[i].Count > skillStats[j].Count
	})

	// Build sorted Group stats: only include used groups (Count > 0), Count desc, Name asc
	groupStats := make([]internal.StatItem, 0, len(groupCountMap))
	for name, count := range groupCountMap {
		if count > 0 {
			groupStats = append(groupStats, internal.StatItem{
				Name:  name,
				Count: count,
			})
		}
	}
	sort.Slice(groupStats, func(i, j int) bool {
		if groupStats[i].Count == groupStats[j].Count {
			return groupStats[i].Name < groupStats[j].Name
		}
		return groupStats[i].Count > groupStats[j].Count
	})

	return &internal.StatsReport{
		Skills: skillStats,
		Groups: groupStats,
	}, nil
}

// GetSkillStats returns only the skill usage statistics.
func (a *App) GetSkillStats() ([]internal.StatItem, error) {
	report, err := a.GetStatsReport()
	if err != nil {
		return nil, err
	}
	return report.Skills, nil
}

// GetGroupStats returns only the group usage statistics.
func (a *App) GetGroupStats() ([]internal.StatItem, error) {
	report, err := a.GetStatsReport()
	if err != nil {
		return nil, err
	}
	return report.Groups, nil
}
