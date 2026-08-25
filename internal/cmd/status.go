package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal"
	"github.com/zoroqi/skill_manager/internal/store"
)

var statusGlobal bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active and inactive skills in the current context",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		ctxCfg, skills, groups, err := application.GetStatus(targetDir, statusGlobal)
		if err != nil {
			return err
		}

		cmd.Printf("Context: %s (%s)\n", ctxCfg.Scope, targetDir)
		cmd.Println("--------------------------------------------------------------------------------")

		// Gather information
		skillMap := make(map[string]internal.Skill)
		for _, s := range skills {
			skillMap[s.ID] = s
		}

		groupMap := make(map[string]internal.Group)
		for _, g := range groups {
			groupMap[g.Name] = g
		}

		existingAgents := store.FindExistingAgents(targetDir)

		// 1. Mark Inactive for exclusion from groups
		inactive := make(map[string]map[string]bool)
		for _, usage := range ctxCfg.InactiveSkills {
			if inactive[usage.Name] == nil {
				inactive[usage.Name] = make(map[string]bool)
			}
			if len(usage.Agents) == 0 {
				inactive[usage.Name][""] = true
			} else {
				for _, a := range usage.Agents {
					inactive[usage.Name][a] = true
				}
			}
		}
		isInactive := func(name, agent string) bool {
			m := inactive[name]
			if m == nil {
				return false
			}
			return m[""] || m[agent]
		}

		// 2. Resolve final skills per agent to track effective state
		// agent -> skillName -> effectiveSkillID
		effectiveID := make(map[string]map[string]string)

		// agent -> skillID -> sources
		idSources := make(map[string]map[string]map[string]bool)

		setEffective := func(agent, skillName, id, source string) {
			if effectiveID[agent] == nil {
				effectiveID[agent] = make(map[string]string)
			}
			if _, exists := effectiveID[agent][skillName]; !exists {
				effectiveID[agent][skillName] = id
			}

			// Only track source if this ID is the effective one for this name
			if effectiveID[agent][skillName] == id {
				if idSources[agent] == nil {
					idSources[agent] = make(map[string]map[string]bool)
				}
				if idSources[agent][id] == nil {
					idSources[agent][id] = make(map[string]bool)
				}
				idSources[agent][id][source] = true
			}
		}

		var missingSkills []internal.SkillUsage
		// Process User Added (Highest priority)
		for _, usage := range ctxCfg.ActiveSkills {
			skill, ok := skillMap[usage.ID]
			if !ok {
				missingSkills = append(missingSkills, usage)
				continue
			}
			agents := usage.Agents
			if len(agents) == 0 {
				agents = existingAgents
			}
			for _, a := range agents {
				setEffective(a, skill.Name, skill.ID, "user")
			}
		}

		// Process Groups (Lower priority, sorted)
		var missingGroups []internal.GroupUsage
		sortedGroups := make([]internal.GroupUsage, len(ctxCfg.ActiveGroups))
		copy(sortedGroups, ctxCfg.ActiveGroups)
		sort.Slice(sortedGroups, func(i, j int) bool {
			return sortedGroups[i].Name < sortedGroups[j].Name
		})

		for _, usage := range sortedGroups {
			group, ok := groupMap[usage.Name]
			if !ok {
				missingGroups = append(missingGroups, usage)
				continue
			}
			agents := usage.Agents
			if len(agents) == 0 {
				agents = existingAgents
			}

			for _, skillID := range group.SkillIDs {
				skill, ok := skillMap[skillID]
				if !ok {
					continue
				}
				for _, a := range agents {
					if isInactive(skill.Name, a) {
						continue
					}

					setEffective(a, skill.Name, skillID, usage.Name)
				}
			}
		}

		// Consolidate for display: Group by SkillID
		type displaySkill struct {
			Skill   internal.Skill
			Agents  map[string]bool
			Sources map[string]bool
		}
		activeDisplay := make(map[string]*displaySkill)

		for agent, skills := range effectiveID {
			for _, id := range skills {
				if activeDisplay[id] == nil {
					activeDisplay[id] = &displaySkill{
						Skill:   skillMap[id],
						Agents:  make(map[string]bool),
						Sources: make(map[string]bool),
					}
				}
				activeDisplay[id].Agents[agent] = true
				for s := range idSources[agent][id] {
					activeDisplay[id].Sources[s] = true
				}
			}
		}

		// Sort by Name
		var sortedIDs []string
		for id := range activeDisplay {
			sortedIDs = append(sortedIDs, id)
		}
		sort.Slice(sortedIDs, func(i, j int) bool {
			return activeDisplay[sortedIDs[i]].Skill.Name < activeDisplay[sortedIDs[j]].Skill.Name
		})

		cmd.Printf("%-15s %-15s %-20s %s\n", "Name", "Register", "Agent", "source")
		if len(sortedIDs) == 0 {
			cmd.Println("  (no active skills)")
		} else {
			for _, id := range sortedIDs {
				d := activeDisplay[id]
				var agents []string
				for a := range d.Agents {
					agents = append(agents, a)
				}
				sort.Strings(agents)

				var sources []string
				for s := range d.Sources {
					sources = append(sources, s)
				}
				sort.Strings(sources)

				reg := d.Skill.RegistryName
				if reg == "" {
					reg = "local"
				}

				cmd.Printf("%-15s %-15s %-20s %s\n",
					d.Skill.Name,
					reg,
					strings.Join(agents, "/"),
					strings.Join(sources, ", "))
			}
		}

		if len(missingSkills) > 0 {
			deletedSkills, _ := application.GetDeletedSkills()
			deletedMap := make(map[string]internal.Skill, len(deletedSkills))
			for _, d := range deletedSkills {
				deletedMap[d.ID] = d
			}

			cmd.Println()
			cmd.Printf("Warning: %d active skill(s) missing or deleted from skills library:\n", len(missingSkills))
			for _, m := range missingSkills {
				agentsStr := "all agents"
				if len(m.Agents) > 0 {
					agentsStr = strings.Join(m.Agents, ", ")
				}
				nameStr := m.ID
				if d, ok := deletedMap[m.ID]; ok && d.Name != "" {
					if d.RegistryName != "" {
						nameStr = fmt.Sprintf("%s (registry: %s, id: %s)", d.Name, d.RegistryName, m.ID)
					} else {
						nameStr = fmt.Sprintf("%s (id: %s)", d.Name, m.ID)
					}
				}
				cmd.Printf("  - %s (agents: %s)\n", nameStr, agentsStr)
			}
		}

		if len(missingGroups) > 0 {
			cmd.Println()
			cmd.Printf("Warning: %d active group(s) missing or deleted from groups library:\n", len(missingGroups))
			for _, m := range missingGroups {
				agentsStr := "all agents"
				if len(m.Agents) > 0 {
					agentsStr = strings.Join(m.Agents, ", ")
				}
				cmd.Printf("  - %s (agents: %s)\n", m.Name, agentsStr)
			}
		}

		if len(missingSkills) > 0 || len(missingGroups) > 0 {
			if statusGlobal {
				cmd.Println("\nPlease run `skill_manager sync -g` to clean up invalid skills.")
			} else {
				cmd.Println("\nPlease run `skill_manager sync` to clean up invalid skills.")
			}
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&statusGlobal, "global", "g", false, "Show global context status")
	rootCmd.AddCommand(statusCmd)
}
