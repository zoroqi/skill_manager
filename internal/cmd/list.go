package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)
		cfg, err := manager.LoadBaseConfig()
		if err != nil {
			return err
		}

		cmd.Printf("%-35s %-20s %s\n", "ID", "Name", "Path")
		for _, s := range cfg.Skills {
			cmd.Printf("%-35s %-20s %s\n", s.ID, s.Name, s.Path)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
