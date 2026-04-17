// internal/cmd/add.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var (
	addGlobal bool
	addAgents string
)

var addCmd = &cobra.Command{
	Use:   "add <skill_name>",
	Short: "Add a managed skill to the current project or globally",
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

		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		var ctxCfg config.ContextConfig
		var saveCtx func() error

		if addGlobal {
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

		// Create symlinks
		for _, a := range agentsToTarget {
			a = strings.TrimSpace(a)
			skillsDir, err := config.EnsureAgentSkillsDir(targetDir, a)
			if err != nil {
				return err
			}

			linkPath := filepath.Join(skillsDir, targetSkill.Name)

			// Remove existing link/file if any
			if _, err := os.Lstat(linkPath); err == nil {
				os.Remove(linkPath)
			}

			if err := os.Symlink(targetSkill.Path, linkPath); err != nil {
				return fmt.Errorf("ftailed to symlink %s to %s: %w", targetSkill.Path, linkPath, err)
			}
			fmt.Printf("Successfully linked %s to %s\n", targetSkill.Name, linkPath)
		}

		if err := saveCtx(); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		fmt.Printf("Skill '%s' added successfully to %s context.\n", skillName, ctxCfg.Scope)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVarP(&addGlobal, "global", "g", false, "Add skill globally")
	addCmd.Flags().StringVar(&addAgents, "agent", "", "Comma-separated list of agents to link (e.g., gemini,claude)")
	rootCmd.AddCommand(addCmd)
}
