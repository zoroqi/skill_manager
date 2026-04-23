package cmd

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := application.ListSkills()
		if err != nil {
			return err
		}

		cmd.Printf("%-35s %-20s %s\n", "ID", "Register", "Name")
		for _, s := range skills {
			cmd.Printf("%-35s %-30s %s\n", s.ID, s.RegistryName, s.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
