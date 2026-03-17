package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-cli",
		Short: "Inspect local observability agents",
	}

	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newDiscoverCommand())
	cmd.AddCommand(newTUICommand())
	cmd.AddCommand(newCompletionCommand(cmd))

	return cmd
}

// Execute runs the root command tree.
func Execute() error {
	return newRootCommand().Execute()
}

func bindPathFlags(cmd *cobra.Command, path *string) {
	cmd.Flags().StringVar(path, "path", "", "Installation or config directory to scan (skip auto-discovery)")
	cmd.Flags().StringVar(path, "config-dir", "", "Alias for --path")
}

func registerAgentFlagCompletion(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("agent", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		choices := []string{"elastic-agent", "edot", "otel"}
		return choices, cobra.ShellCompDirectiveNoFileComp
	})
}

func newCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch strings.ToLower(strings.TrimSpace(args[0])) {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}
