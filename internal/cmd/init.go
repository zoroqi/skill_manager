package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)
		if err := manager.Init(); err != nil {
			return err
		}
		fmt.Printf("Configuration initialized at %s\n", configDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
