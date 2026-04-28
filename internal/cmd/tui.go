package cmd

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/zoroqi/skill_manager/internal/tui"
	"os"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the TUI interface",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(tui.NewModel(application), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
