package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	removeGlobal bool
	removeGroup  string
	removeAgents string
)

var removeCmd = &cobra.Command{
	Use:   "remove [skill_name]",
	Short: "Remove a skill or group from the current context and update symlinks",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		if removeGroup != "" {
			var agents []string
			if removeAgents != "" {
				agents = strings.Split(removeAgents, ",")
			}
			if err := application.RemoveGroup(targetDir, removeGroup, agents, removeGlobal); err != nil {
				return err
			}
			fmt.Printf("Group '%s' removed and synced successfully.\n", removeGroup)
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("please specify a skill name or use --group flag")
		}

		if err := application.RemoveSkill(targetDir, args[0], removeGlobal); err != nil {
			return err
		}

		scope := "local"
		if removeGlobal {
			scope = "global"
		}
		fmt.Printf("Skill '%s' removed and synced successfully from %s context.\n", args[0], scope)
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeGlobal, "global", "g", false, "Remove skill from global context")
	removeCmd.Flags().StringVar(&removeGroup, "group", "", "Name of the group to remove")
	removeCmd.Flags().StringVar(&removeAgents, "agent", "", "Comma-separated list of agents to remove from group")
	rootCmd.AddCommand(removeCmd)
}
