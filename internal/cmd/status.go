// internal/cmd/status.go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var statusGlobal bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show active and inactive skills in the current context",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		baseCfg, err := manager.LoadBaseConfig()
		if err != nil {
			return fmt.Errorf("failed to load base config: %w", err)
		}

		// Map skill ID to skill object for easy lookup
		skillMap := make(map[string]config.Skill)
		for _, s := range baseCfg.Skills {
			skillMap[s.ID] = s
		}

		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		var ctxCfg config.ContextConfig
		if statusGlobal {
			ctxCfg, err = manager.LoadContextConfig(config.FILENAME_GLOBAL_CONTEXT)
			if err != nil {
				return fmt.Errorf("failed to load global context: %w", err)
			}
		} else {
			ctxCfg, err = config.EnsureLocalContext(targetDir)
			if err != nil {
				return fmt.Errorf("failed to load local context: %w", err)
			}
		}

		cmd.Printf("Context: %s (%s)\n", ctxCfg.Scope, targetDir)
		cmd.Println("------------------------------------------------------------")

		cmd.Println("Active Skills:")
		if len(ctxCfg.ActiveSkills) == 0 {
			cmd.Println("  (none)")
		} else {
			cmd.Printf("  %-20s %-20s %s\n", "Name", "Agents", "Path")
			for _, usage := range ctxCfg.ActiveSkills {
				if s, ok := skillMap[usage.ID]; ok {
					cmd.Printf("  %-20s %-20s %s\n", s.Name, strings.Join(usage.Agents, ","), s.Path)
				} else {
					cmd.Printf("  %-20s (missing from scanned library)\n", usage.ID)
				}
			}
		}

		cmd.Println("\nInactive Skills:")
		if len(ctxCfg.InactiveSkills) == 0 {
			cmd.Println("  (none)")
		} else {
			cmd.Printf("  %-20s %-20s %s\n", "Name", "Agents", "Path")
			for _, usage := range ctxCfg.InactiveSkills {
				if s, ok := skillMap[usage.ID]; ok {
					cmd.Printf("  %-20s %-20s %s\n", s.Name, strings.Join(usage.Agents, ","), s.Path)
				} else {
					cmd.Printf("  %-20s (missing from scanned library)\n", usage.ID)
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
