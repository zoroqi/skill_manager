// internal/cmd/remove.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var removeGlobal bool

var removeCmd = &cobra.Command{
	Use:   "remove <skill_name>",
	Short: "Remove a skill from the current context and delete symlinks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]

		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

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

		// Cleanup symlinks
		if err := config.RemoveAgentSymlinks(targetDir, skillName, nil); err != nil {
			return err
		}

		// Remove from context
		manager.RemoveSkillFromContext(&ctxCfg, skillID)

		if err := saveCtx(); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		fmt.Printf("Skill '%s' removed successfully from %s context.\n", skillName, ctxCfg.Scope)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeGlobal, "global", "g", false, "Remove skill from global context")
	rootCmd.AddCommand(removeCmd)
}
