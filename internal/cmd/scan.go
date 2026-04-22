package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [dir]",
	Short: "Scan directory for skills and update config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		count, err := application.ScanSkills(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Scanned and added %d skills.\n", count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
