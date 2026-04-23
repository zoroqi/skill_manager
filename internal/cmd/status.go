package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
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

		skillMap := make(map[string]string)
		for _, s := range skills {
			skillMap[s.ID] = s.Name
		}
		skillRegisterMap := make(map[string]string)
		for _, s := range skills {
			skillRegisterMap[s.ID] = s.RegistryName
		}

		cmd.Printf("Context: %s (%s)\n", ctxCfg.Scope, targetDir)
		cmd.Println("------------------------------------------------------------")

		cmd.Println("Active Skills:")
		if len(ctxCfg.ActiveSkills) == 0 {
			cmd.Println("  (none)")
		} else {
			cmd.Printf("  %-20s %-20s %s\n", "Name", "Agents", "Register")
			for _, usage := range ctxCfg.ActiveSkills {
				if name, ok := skillMap[usage.ID]; ok {
					cmd.Printf("  %-20s %-20s %s\n", name, strings.Join(usage.Agents, ","), skillRegisterMap[usage.ID])
				} else {
					cmd.Printf("  %-20s (missing from scanned library)\n", usage.ID)
				}
			}
		}

		cmd.Println("\nInactive Skills:")
		if len(ctxCfg.InactiveSkills) == 0 {
			cmd.Println("  (none)")
		} else {
			cmd.Printf("  %-20s %-20s %s\n", "Name", "Agents", "Register")
			for _, usage := range ctxCfg.InactiveSkills {
				if name, ok := skillMap[usage.ID]; ok {
					cmd.Printf("  %-20s %-20s %s\n", name, strings.Join(usage.Agents, ","), skillRegisterMap[usage.ID])
				} else {
					cmd.Printf("  %-20s (missing from scanned library)\n", usage.ID)
				}
			}
		}

		groupsSkillsMap := make(map[string][]string)
		for _, g := range groups {
			groupsSkillsMap[g.Name] = append(groupsSkillsMap[g.Name], g.SkillIDs...)
		}

		cmd.Println("\nGrouped Skills:")
		if len(ctxCfg.ActiveGroups) == 0 {
			cmd.Println("  (none)")
		} else {
			cmd.Printf("  %-20s %-20s %s\n", "Name", "Agents", "Register")
			for _, usage := range ctxCfg.ActiveGroups {
				if group, ok := groupsSkillsMap[usage]; ok {
					for _, id := range group {
						if name, ok := skillMap[id]; ok {
							cmd.Printf("  %-20s %-20s %s\n", name, "", skillRegisterMap[id])
						} else {
							cmd.Printf("  %-20s (missing from scanned library)\n", id)
						}
					}
				}
			}
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVarP(&statusGlobal, "global", "g", false, "Show global context status")
	rootCmd.AddCommand(statusCmd)
}
