// internal/cmd/stop.go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var (
	stopGlobal bool
	stopAgents string
)

var stopCmd = &cobra.Command{
	Use:   "stop <skill_name>",
	Short: "Stop using a skill (deactivate) in the current context",
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

		if stopGlobal {
			targetDir, err = os.UserHomeDir()
			if err != nil {
				return err
			}
			ctxCfg, err = manager.LoadContextConfig(config.FILENAME_GLOBAL_CONTEXT)
			if err != nil {
				return fmt.Errorf("failed to load global context: %w", err)
			}
			saveCtx = func() error {
				return manager.SaveContextConfig(ctxCfg, config.FILENAME_GLOBAL_CONTEXT)
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
		agentsToStop := []string{}
		if stopAgents != "" {
			agentsToStop = strings.Split(stopAgents, ",")
		}
		if err := config.RemoveAgentSymlinks(targetDir, skillName, agentsToStop); err != nil {
			return err
		}

		// Move to inactive in context
		manager.StopSkillInContext(&ctxCfg, skillID, agentsToStop)

		if err := saveCtx(); err != nil {
			return fmt.Errorf("failed to save context: %w", err)
		}

		fmt.Printf("Skill '%s' stopped successfully in %s context.\n", skillName, ctxCfg.Scope)
		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVarP(&stopGlobal, "global", "g", false, "Stop skill in global context")
	stopCmd.Flags().StringVar(&stopAgents, "agent", "", "Comma-separated list of agents to stop (e.g., gemini,claude)")
	rootCmd.AddCommand(stopCmd)
}
