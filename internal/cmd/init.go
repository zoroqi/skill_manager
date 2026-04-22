package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := application.Initialize(); err != nil {
			return err
		}
		fmt.Println("Configuration initialized.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
