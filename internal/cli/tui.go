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
	var otelConfigPath string
	var otelZPagesURL string
	var otelMetricsURL string
	var otelHealthURL string
	var installPath string

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipe, err := statusPipeline(cmd, statusOptions{
				agentType:         agentType,
				elasticConfig:     elasticConfigPath,
				elasticStatusURL:  elasticStatusURL,
				elasticURLSet:     cmd.Flags().Changed("elastic-url"),
				edotConfig:        edotConfigPath,
				edotZPagesURL:     edotZPagesURL,
				edotZPagesURLSet:  cmd.Flags().Changed("edot-zpages-url"),
				edotMetricsURL:    edotMetricsURL,
				edotMetricsURLSet: cmd.Flags().Changed("edot-metrics-url"),
				edotHealthURL:     edotHealthURL,
				edotHealthURLSet:  cmd.Flags().Changed("edot-health-url"),
				otelConfig:        otelConfigPath,
				otelZPagesURL:     otelZPagesURL,
				otelZPagesURLSet:  cmd.Flags().Changed("otel-zpages-url"),
				otelMetricsURL:    otelMetricsURL,
				otelMetricsURLSet: cmd.Flags().Changed("otel-metrics-url"),
				otelHealthURL:     otelHealthURL,
				otelHealthURLSet:  cmd.Flags().Changed("otel-health-url"),
				path:              installPath,
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
	cmd.Flags().DurationVar(&refresh, "refresh", 0, "Refresh interval in live mode (enables live mode when set)")
	cmd.Flags().StringVar(&agentType, "agent", "", "Target a specific agent type")
	cmd.Flags().StringVar(&elasticConfigPath, "elastic-config", "", "Path to elastic-agent.yml (auto-detected when omitted)")
	cmd.Flags().StringVar(&elasticStatusURL, "elastic-url", "http://localhost:6791", "Elastic Agent status API base URL")
	cmd.Flags().StringVar(&edotConfigPath, "edot-config", "", "Path to EDOT/OTel collector YAML config")
	cmd.Flags().StringVar(&edotZPagesURL, "edot-zpages-url", "http://localhost:55679", "EDOT zpages base URL")
	cmd.Flags().StringVar(&edotMetricsURL, "edot-metrics-url", "http://localhost:8888/metrics", "EDOT Prometheus metrics endpoint")
	cmd.Flags().StringVar(&edotHealthURL, "edot-health-url", "http://localhost:13133/", "EDOT health_check endpoint")
	cmd.Flags().StringVar(&otelConfigPath, "otel-config", "", "Path to OTel collector YAML config")
	cmd.Flags().StringVar(&otelZPagesURL, "otel-zpages-url", "http://localhost:55679", "OTel zpages base URL")
	cmd.Flags().StringVar(&otelMetricsURL, "otel-metrics-url", "http://localhost:8888/metrics", "OTel Prometheus metrics endpoint")
	cmd.Flags().StringVar(&otelHealthURL, "otel-health-url", "http://localhost:13133/", "OTel health_check endpoint")
	bindPathFlags(cmd, &installPath)
	registerAgentFlagCompletion(cmd)

	return cmd
}
