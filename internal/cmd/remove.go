// internal/cmd/remove.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var (
	removeGlobal bool
	removeGroup  string
)

var removeCmd = &cobra.Command{
	Use:   "remove [skill_name]",
	Short: "Remove a skill or group from the current context and update symlinks",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		var ctxCfg config.ContextConfig
		var saveCtx func() error

		if removeGlobal {
			targetDir, err = os.UserHomeDir()
			if err != nil {
				return err
			}
			ctxCfg, err = manager.LoadContextConfig("global_context.json")
			if err != nil {
				return fmt.Errorf("failed to load global context: %w", err)
			}
			saveCtx = func() error {
				return manager.SaveContextConfig(ctxCfg, "global_context.json")
			}
		} else {
			ctxCfg, err = config.EnsureLocalContext(targetDir)
			if err != nil {
				return fmt.Errorf("failed to load local context: %w", err)
			}
			saveCtx = func() error {
				return config.SaveLocalContext(targetDir, ctxCfg)
			}
		}

		if removeGroup != "" {
			// Remove group from context
			newGroups := []string{}
			found := false
			for _, g := range ctxCfg.ActiveGroups {
				if g == removeGroup {
					found = true
					continue
				}
				newGroups = append(newGroups, g)
			}
			if !found {
				return fmt.Errorf("group '%s' not active in this context", removeGroup)
			}
			ctxCfg.ActiveGroups = newGroups

			// Sync to remove symlinks
			if err := manager.SyncContext(targetDir, &ctxCfg, true); err != nil {
				return err
			}

			if err := saveCtx(); err != nil {
				return err
			}
			fmt.Printf("Group '%s' removed and synced successfully.\n", removeGroup)
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("please specify a skill name or use --group flag")
		}
		skillName := args[0]

		cfg, err := manager.LoadBaseConfig()
		if err != nil {
			return fmt.Errorf("failed to load base config: %w", err)
		}

		// Find the skill ID
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

		// Remove from context
		manager.RemoveSkillFromContext(&ctxCfg, skillID)

		// Sync to remove symlinks
		if err := manager.SyncContext(targetDir, &ctxCfg, true); err != nil {
			return err
		}

		if err := saveCtx(); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		fmt.Printf("Skill '%s' removed and synced successfully from %s context.\n", skillName, ctxCfg.Scope)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeGlobal, "global", "g", false, "Remove skill from global context")
	removeCmd.Flags().StringVar(&removeGroup, "group", "", "Name of the group to remove")
	rootCmd.AddCommand(removeCmd)
}
