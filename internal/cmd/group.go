package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/config"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage skill groups (packs)",
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name> [description]",
	Short: "Create a new empty group",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		description := ""
		if len(args) > 1 {
			description = strings.Join(args[1:], " ")
		}

		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		if err := manager.CreateGroup(name, description); err != nil {
			return err
		}
		fmt.Printf("Group '%s' created successfully.\n", name)
		return nil
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		if err := manager.DeleteGroup(name); err != nil {
			return err
		}
		fmt.Printf("Group '%s' deleted successfully.\n", name)
		return nil
	},
}

var groupAddSkillCmd = &cobra.Command{
	Use:   "add-skill <group_name> <skill_name>...",
	Short: "Add one or more skills to a group",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName := args[0]
		skillNames := args[1:]

		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		baseCfg, err := manager.LoadBaseConfig()
		if err != nil {
			return err
		}

		for _, sn := range skillNames {
			var skillID string
			for _, s := range baseCfg.Skills {
				if s.Name == sn {
					skillID = s.ID
					break
				}
			}
			if skillID == "" {
				return fmt.Errorf("skill '%s' not found", sn)
			}
			if err := manager.AddSkillToGroup(groupName, skillID); err != nil {
				return err
			}
			fmt.Printf("Skill '%s' added to group '%s'.\n", sn, groupName)
		}
		return nil
	},
}

var groupRemoveSkillCmd = &cobra.Command{
	Use:   "remove-skill <group_name> <skill_name>...",
	Short: "Remove one or more skills from a group",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName := args[0]
		skillNames := args[1:]

		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		baseCfg, err := manager.LoadBaseConfig()
		if err != nil {
			return err
		}

		for _, sn := range skillNames {
			var skillID string
			for _, s := range baseCfg.Skills {
				if s.Name == sn {
					skillID = s.ID
					break
				}
			}
			if skillID == "" {
				return fmt.Errorf("skill '%s' not found in base config, skipping", sn)
			}
			if err := manager.RemoveSkillFromGroup(groupName, skillID); err != nil {
				return err
			}
			fmt.Printf("Skill '%s' removed from group '%s'.\n", sn, groupName)
		}
		return nil
	},
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all groups and their skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		configDir, err := config.GetDefaultConfigDir()
		if err != nil {
			return err
		}
		manager := config.NewManager(configDir)

		groupsCfg, err := manager.LoadGroupsConfig()
		if err != nil {
			return err
		}
		baseCfg, err := manager.LoadBaseConfig()
		if err != nil {
			return err
		}

		skillMap := make(map[string]string)
		for _, s := range baseCfg.Skills {
			skillMap[s.ID] = s.Name
		}

		fmt.Println("Skill Groups:")
		for _, g := range groupsCfg.Groups {
			skillNames := []string{}
			for _, id := range g.SkillIDs {
				if name, ok := skillMap[id]; ok {
					skillNames = append(skillNames, name)
				} else {
					skillNames = append(skillNames, fmt.Sprintf("Unknown(%s)", id))
				}
			}
			fmt.Printf("- %s: %s\n  Skills: [%s]\n", g.Name, g.Description, strings.Join(skillNames, ", "))
		}
		return nil
	},
}

func init() {
	groupCmd.AddCommand(groupCreateCmd)
	groupCmd.AddCommand(groupDeleteCmd)
	groupCmd.AddCommand(groupAddSkillCmd)
	groupCmd.AddCommand(groupRemoveSkillCmd)
	groupCmd.AddCommand(groupListCmd)
	rootCmd.AddCommand(groupCmd)
}
