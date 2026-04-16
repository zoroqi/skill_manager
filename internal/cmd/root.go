package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "skill_manager",
	Short: "A CLI tool for managing AI Agent skills",
	Long: `A CLI tool for managing AI Agent skills across different projects and globally.
Config files are stored in the directory specified by the SKILL_MANAGER_CONFIG 
environment variable. If not set, it defaults to ~/.skill_manager.`,
}

func Execute() error {
	return rootCmd.Execute()
}
