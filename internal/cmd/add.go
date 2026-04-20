// internal/cmd/add.go
package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var (
	addGlobal bool
	addAgents string
	addGroup  string
	addForce  bool
)

var addCmd = &cobra.Command{
	Use:   "add [skill_name]",
	Short: "Add a managed skill or group to the current project or globally",
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
		var configFilename string

		if addGlobal {
			targetDir, err = os.UserHomeDir()
			if err != nil {
				return err
			}
			configFilename = "global_context.json"
			ctxCfg, err = manager.LoadContextConfig(configFilename)
			if err != nil {
				return fmt.Errorf("failed to load global context: %w", err)
			}
			saveCtx = func() error {
				return manager.SaveContextConfig(ctxCfg, configFilename)
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

		if addGroup != "" {
			// Add a group
			group, err := manager.GetGroup(addGroup)
			if err != nil {
				return err
			}
			if group == nil {
				return fmt.Errorf("group '%s' not found", addGroup)
			}

			// Add to ActiveGroups if not already there
			found := slices.Contains(ctxCfg.ActiveGroups, addGroup)
			if !found {
				ctxCfg.ActiveGroups = append(ctxCfg.ActiveGroups, addGroup)
			}

			if err := manager.SyncContext(targetDir, &ctxCfg, addForce); err != nil {
				return err
			}
			if err := saveCtx(); err != nil {
				return err
			}
			fmt.Printf("Group '%s' added and synced successfully.\n", addGroup)
			return nil
		}

		// Original add skill logic
		if len(args) == 0 {
			return fmt.Errorf("please specify a skill name or use --group flag")
		}
		skillName := args[0]

		cfg, err := manager.LoadBaseConfig()
		if err != nil {
			return fmt.Errorf("failed to load base config: %w", err)
		}

		// Find the skill
		var targetSkill *config.Skill
		for _, s := range cfg.Skills {
			if s.Name == skillName {
				targetSkill = &s
				break
			}
		}

		if targetSkill == nil {
			return fmt.Errorf("skill '%s' not found. please scan it first", skillName)
		}

		var agentsToTarget []string
		if addAgents != "" {
			agentsToTarget = strings.Split(addAgents, ",")
		} else {
			agentsToTarget = config.FindExistingAgents(targetDir)
			if len(agentsToTarget) == 0 {
				return fmt.Errorf("no agent directories found. Please specify --agent flag (e.g. --agent gemini) to create one")
			}
		}

		manager.AddSkillToContext(&ctxCfg, targetSkill.ID, agentsToTarget)

		if err := manager.SyncContext(targetDir, &ctxCfg, addForce); err != nil {
			return err
		}

		if err := saveCtx(); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		fmt.Printf("Skill '%s' added and synced successfully to %s context.\n", skillName, ctxCfg.Scope)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVarP(&addGlobal, "global", "g", false, "Add skill globally")
	addCmd.Flags().StringVar(&addAgents, "agent", "", "Comma-separated list of agents to link (e.g., gemini,claude)")
	addCmd.Flags().StringVar(&addGroup, "group", "", "Name of the group to add")
	addCmd.Flags().BoolVarP(&addForce, "force", "f", false, "Force overwrite existing symlinks")
	rootCmd.AddCommand(addCmd)
}
