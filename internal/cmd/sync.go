package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var (
	syncGlobal bool
	syncForce  bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize current context's symlinks based on active skills and groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		targetDir, err := os.Getwd()
		if err != nil {
			return err
		}

		var ctxCfg config.ContextConfig
		if syncGlobal {
			targetDir, err = os.UserHomeDir()
			if err != nil {
				return err
			}
			ctxCfg, err = manager.LoadContextConfig(config.FILENAME_GLOBAL_CONTEXT)
			if err != nil {
				return fmt.Errorf("failed to load global context: %w", err)
			}
		} else {
			ctxCfg, err = config.EnsureLocalContext(targetDir)
			if err != nil {
				return fmt.Errorf("failed to load local context: %w", err)
			}
		}

		if err := manager.SyncContext(targetDir, &ctxCfg, syncForce); err != nil {
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
