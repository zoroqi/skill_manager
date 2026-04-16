package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
	"github.com/zoroqi/skill_manager/internal/scanner"
)

var scanCmd = &cobra.Command{
	Use:   "scan [dir]",
	Short: "Scan directory for skills and update config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := args[0]
		skills, err := scanner.Scan(targetDir)
		if err != nil {
			return err
		}

		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)
		cfg, err := manager.LoadBaseConfig()
		if err != nil {
			return err
		}

		// Merge new skills avoiding duplicates by ID
		skillMap := make(map[string]config.Skill)
		for _, s := range cfg.Skills {
			skillMap[s.ID] = s
		}
		for _, s := range skills {
			skillMap[s.ID] = s
		}

		newSkills := make([]config.Skill, 0, len(skillMap))
		for _, s := range skillMap {
			newSkills = append(newSkills, s)
		}
		cfg.Skills = newSkills

		if err := manager.SaveBaseConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("Scanned and added %d skills.\n", len(skills))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
