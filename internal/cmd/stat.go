package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal"
)

var statCmd = &cobra.Command{
	Use:     "stat",
	Aliases: []string{"stats", "statistics"},
	Short:   "Show usage statistics for skills and groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := application.GetStatsReport()
		if err != nil {
			return err
		}

		printSkillStats(cmd, report.Skills)
		cmd.Println()
		printGroupStats(cmd, report.Groups)
		return nil
	},
}

var statSkillCmd = &cobra.Command{
	Use:     "skill",
	Aliases: []string{"skills"},
	Short:   "Show usage statistics for individual skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := application.GetSkillStats()
		if err != nil {
			return err
		}
		printSkillStats(cmd, skills)
		return nil
	},
}

var statGroupCmd = &cobra.Command{
	Use:     "group",
	Aliases: []string{"groups"},
	Short:   "Show usage statistics for skill groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		groups, err := application.GetGroupStats()
		if err != nil {
			return err
		}
		printGroupStats(cmd, groups)
		return nil
	},
}

func printStatTable(cmd *cobra.Command, titleName string, items []internal.StatItem) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 4, ' ', 0)
	fmt.Fprintf(w, "%s\tCount\n", titleName)
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%d\n", item.Name, item.Count)
	}
	_ = w.Flush()
}

func printSkillStats(cmd *cobra.Command, skills []internal.StatItem) {
	printStatTable(cmd, "Skill Name", skills)
}

func printGroupStats(cmd *cobra.Command, groups []internal.StatItem) {
	printStatTable(cmd, "Group Name", groups)
}

func init() {
	statCmd.AddCommand(statSkillCmd)
	statCmd.AddCommand(statGroupCmd)
	rootCmd.AddCommand(statCmd)
}
