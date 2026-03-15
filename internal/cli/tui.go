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
	var edotConfigPath string
	var edotZPagesURL string
	var edotMetricsURL string
	var edotHealthURL string

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipe, err := statusPipeline(cmd, statusOptions{
				agentType:        agentType,
				elasticConfig:    elasticConfigPath,
				elasticStatusURL: elasticStatusURL,
				edotConfig:       edotConfigPath,
				edotZPagesURL:    edotZPagesURL,
				edotMetricsURL:   edotMetricsURL,
				edotHealthURL:    edotHealthURL,
			})
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
	cmd.Flags().StringVar(&elasticConfigPath, "elastic-config", "", "Path to elastic-agent.yml (auto-detected when omitted)")
	cmd.Flags().StringVar(&elasticStatusURL, "elastic-url", "http://localhost:6791", "Elastic Agent status API base URL")
	cmd.Flags().StringVar(&edotConfigPath, "edot-config", "", "Path to EDOT/OTel collector YAML config")
	cmd.Flags().StringVar(&edotZPagesURL, "edot-zpages-url", "http://localhost:55679", "EDOT zpages base URL")
	cmd.Flags().StringVar(&edotMetricsURL, "edot-metrics-url", "http://localhost:8888/metrics", "EDOT Prometheus metrics endpoint")
	cmd.Flags().StringVar(&edotHealthURL, "edot-health-url", "http://localhost:13133/", "EDOT health_check endpoint")

	return cmd
}
