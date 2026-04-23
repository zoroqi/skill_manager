package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/app"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "skill_manager",
	Short: "A CLI tool for managing AI Agent skills",
	Long: `A CLI tool for managing AI Agent skills across different projects and globally.
Config files are stored in the directory specified by the SKILL_MANAGER_CONFIG 
environment variable. If not set, it defaults to ~/.skill_manager.`,
}

var application *app.App

func init() {
	cobra.OnInitialize(func() {
		configDir, err := app.DefaultConfigDir()
		if err != nil {
			panic(err)
		}
		application = app.New(configDir)
	})
}

func Execute() error {
	rootCmd.SetOut(os.Stdout)
	return rootCmd.Execute()
}
