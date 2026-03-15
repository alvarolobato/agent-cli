package cli

import (
	"time"

	"github.com/alvarolobato/agent-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func newTUICommand() *cobra.Command {
	var live bool
	var refresh time.Duration

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			program := tea.NewProgram(tui.NewModel(live, refresh))
			_, err := program.Run()
			return err
		},
	}

	cmd.Flags().BoolVar(&live, "live", false, "Enable live mode auto-refresh")
	cmd.Flags().DurationVar(&refresh, "refresh", 5*time.Second, "Refresh interval in live mode")

	return cmd
}
