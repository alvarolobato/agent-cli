package cli

import "github.com/spf13/cobra"

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-cli",
		Short: "Inspect local observability agents",
	}

	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newDiscoverCommand())
	cmd.AddCommand(newTUICommand())

	return cmd
}

// Execute runs the root command tree.
func Execute() error {
	return newRootCommand().Execute()
}
