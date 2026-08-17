package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal"
)

var summaryCmd = &cobra.Command{
	Use:     "summary",
	Aliases: []string{"summaries"},
	Short:   "Show summary of skills and groups configured across all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := application.GetSummaryReport()
		if err != nil {
			return err
		}

		printSummaryTable(cmd, report)
		return nil
	},
}

func printSummaryTable(cmd *cobra.Command, report *internal.SummaryReport) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 4, ' ', 0)
	fmt.Fprintf(w, "Project\tSkill\tGroup\n")
	for _, item := range report.Items {
		rows := max(len(item.Groups), len(item.Skills))
		if rows == 0 {
			rows = 1
		}

		for i := 0; i < rows; i++ {
			proj := ""
			if i == 0 {
				proj = item.ProjectName
			}
			skill := ""
			if i < len(item.Skills) {
				skill = item.Skills[i]
			}
			group := ""
			if i < len(item.Groups) {
				group = item.Groups[i]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", proj, skill, group)
		}
	}
	_ = w.Flush()
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}
