package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
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

		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		var agents []string
		if stopAgents != "" {
			agents = strings.Split(stopAgents, ",")
		}

		if err := application.StopSkill(targetDir, skillName, agents, stopGlobal); err != nil {
			return err
		}

		scope := "local"
		if stopGlobal {
			scope = "global"
		}
		fmt.Printf("Skill '%s' stopped successfully in %s context.\n", skillName, scope)
		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVarP(&stopGlobal, "global", "g", false, "Stop skill in global context")
	stopCmd.Flags().StringVar(&stopAgents, "agent", "", "Comma-separated list of agents to stop (e.g., gemini,claude)")
	rootCmd.AddCommand(stopCmd)
}
