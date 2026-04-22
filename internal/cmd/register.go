package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	registerName string
	registerPath string
	deleteFiles  bool
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Manage remote skill repositories",
}

var registerAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Clone and register a remote skill repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		err := application.RegisterAdd(url, registerName, registerPath)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully registered and scanned %s\n", url)
		return nil
	},
}

var registerUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Pull updates for a registered repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		err := application.RegisterUpdate(name)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully updated %s\n", name)
		return nil
	},
}

var registerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		registries, err := application.RegisterList()
		if err != nil {
			return err
		}
		if len(registries) == 0 {
			fmt.Println("No registries found.")
			return nil
		}
		fmt.Printf("%-20s %-40s %-30s %-20s\n", "NAME", "URL", "LOCAL PATH", "LAST UPDATED")
		for _, r := range registries {
			fmt.Printf("%-20s %-40s %-30s %-20s\n", r.Name, r.URL, r.LocalPath, r.LastUpdated)
		}
		return nil
	},
}

var registerRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Unregister a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		err := application.RegisterRemove(name, deleteFiles)
		if err != nil {
			return err
		}
		fmt.Printf("Successfully removed %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
	registerCmd.AddCommand(registerAddCmd)
	registerCmd.AddCommand(registerUpdateCmd)
	registerCmd.AddCommand(registerListCmd)
	registerCmd.AddCommand(registerRemoveCmd)

	registerAddCmd.Flags().StringVarP(&registerName, "name", "n", "", "Custom name for the registry")
	registerAddCmd.Flags().StringVarP(&registerPath, "path", "p", "", "Custom path to clone the repository")
	registerRemoveCmd.Flags().BoolVarP(&deleteFiles, "delete", "d", false, "Delete local files (moves to backup)")
}
