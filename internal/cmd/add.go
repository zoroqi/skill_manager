package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	addGlobal   bool
	addAgents   string
	addRegister string
	addGroup    string
	addForce    bool
)

var addCmd = &cobra.Command{
	Use:   "add [skill_name]",
	Short: "Add a managed skill or group to the current project or globally",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		if addGroup != "" {
			var agents []string
			if addAgents != "" {
				agents = strings.Split(addAgents, ",")
			}
			if err := application.AddGroup(targetDir, addGroup, agents, addGlobal, addForce); err != nil {
				return err
			}
			fmt.Printf("Group '%s' added and synced successfully.\n", addGroup)
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("please specify a skill name or use --group flag")
		}

		var agents []string
		if addAgents != "" {
			agents = strings.Split(addAgents, ",")
		}

		if err := application.AddSkill(targetDir, args[0], addRegister, agents, addGlobal, addForce); err != nil {
			return err
		}

		scope := "local"
		if addGlobal {
			scope = "global"
		}
		fmt.Printf("Skill '%s' added and synced successfully to %s context.\n", args[0], scope)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVarP(&addGlobal, "global", "g", false, "Add skill globally")
	addCmd.Flags().StringVar(&addAgents, "agent", "", "Comma-separated list of agents to link (e.g., gemini,claude)")
	addCmd.Flags().StringVarP(&addRegister, "register", "r", "", "Specify the registry name if there are multiple skills with the same name")
	addCmd.Flags().StringVar(&addGroup, "group", "", "Name of the group to add")
	addCmd.Flags().BoolVarP(&addForce, "force", "f", false, "Force overwrite existing symlinks")
	rootCmd.AddCommand(addCmd)
}
