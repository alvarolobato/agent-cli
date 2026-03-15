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
	var agentType string
	var elasticConfigPath string
	var elasticStatusURL string

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipe, err := statusPipeline(cmd, agentType, elasticConfigPath, elasticStatusURL)
			if err != nil {
				return err
			}
			program := tea.NewProgram(tui.NewModel(live, refresh, pipe))
			_, err = program.Run()
			return err
		},
	}

	cmd.Flags().BoolVar(&live, "live", false, "Enable live mode auto-refresh")
	cmd.Flags().DurationVar(&refresh, "refresh", 5*time.Second, "Refresh interval in live mode")
	cmd.Flags().StringVar(&agentType, "agent", "", "Target a specific agent type")
	cmd.Flags().StringVar(&elasticConfigPath, "elastic-config", "test/fixtures/elastic-agent.yml", "Path to elastic-agent.yml")
	cmd.Flags().StringVar(&elasticStatusURL, "elastic-url", "http://localhost:6791", "Elastic Agent status API base URL")

	return cmd
}
