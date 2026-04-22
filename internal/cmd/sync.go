package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	syncGlobal bool
	syncForce  bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize current context's symlinks based on active skills and groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		if err := application.SyncContext(targetDir, syncGlobal, syncForce); err != nil {
			return err
		}

		fmt.Println("Context synchronized successfully.")
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVarP(&syncGlobal, "global", "g", false, "Sync global context")
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Force overwrite existing symlinks")
	rootCmd.AddCommand(syncCmd)
}
