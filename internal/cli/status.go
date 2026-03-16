package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/alvarolobato/agent-cli/internal/agent/edot"
	"github.com/alvarolobato/agent-cli/internal/agent/elasticagent"
	"github.com/alvarolobato/agent-cli/internal/agent/otel"
	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/discovery"
	"github.com/alvarolobato/agent-cli/internal/metrics"
	"github.com/alvarolobato/agent-cli/internal/output"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var agentType string
	var format string
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

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a pipeline-oriented status report",
		RunE: func(cmd *cobra.Command, args []string) error {
			model, err := statusPipeline(cmd, statusOptions{
				agentType:        agentType,
				elasticConfig:    elasticConfigPath,
				elasticStatusURL: elasticStatusURL,
				edotConfig:       edotConfigPath,
				edotZPagesURL:    edotZPagesURL,
				edotMetricsURL:   edotMetricsURL,
				edotHealthURL:    edotHealthURL,
				otelConfig:       otelConfigPath,
				otelZPagesURL:    otelZPagesURL,
				otelMetricsURL:   otelMetricsURL,
				otelHealthURL:    otelHealthURL,
			})
			if err != nil {
				return err
			}

			switch format {
			case "json":
				out, err := output.RenderJSON(model)
				if err != nil {
					return err
				}
				cmd.Println(out)
				return nil
			case "table":
				cmd.Println(output.RenderTable(model))
				return nil
			default:
				return errors.New("unsupported --format value (use: table|json)")
			}
		},
	}

	cmd.Flags().StringVar(&agentType, "agent", "", "Target a specific agent type")
	cmd.Flags().StringVar(&format, "format", "table", fmt.Sprintf("Output format (%s)", "table|json"))
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

	return cmd
}

type statusOptions struct {
	agentType        string
	elasticConfig    string
	elasticStatusURL string
	edotConfig       string
	edotZPagesURL    string
	edotMetricsURL   string
	edotHealthURL    string
	otelConfig       string
	otelZPagesURL    string
	otelMetricsURL   string
	otelHealthURL    string
}

var discoverAgents = func(ctx context.Context) ([]discovery.DiscoveredAgent, error) {
	return discovery.NewOrchestrator().DiscoverDetailed(ctx)
}

func statusPipeline(cmd *cobra.Command, options statusOptions) (*pipeline.Pipeline, error) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if options.agentType == "" {
		var err error
		options, err = autoDetectStatusOptions(ctx, options)
		if err != nil {
			return nil, err
		}
	}

	switch options.agentType {
	case "elastic-agent":
		configPath, err := resolveElasticConfigPath(options.elasticConfig)
		if err != nil {
			return nil, err
		}
		cfg, err := config.ParseElasticAgentConfig(configPath)
		if err != nil {
			return nil, err
		}

		httpClient := &http.Client{Timeout: 5 * time.Second}
		client := elasticagent.NewClient(options.elasticStatusURL, httpClient)
		adapter := elasticagent.NewAdapter(cfg, client)
		return adapter.Status(ctx)
	case "edot":
		if strings.TrimSpace(options.edotConfig) == "" {
			return nil, errors.New("edot config not found; pass --edot-config")
		}

		cfg, err := config.ParseOTelCollectorConfig(options.edotConfig)
		if err != nil {
			return nil, err
		}

		httpClient := &http.Client{Timeout: 5 * time.Second}
		zpagesClient := edot.NewZPagesClient(options.edotZPagesURL, httpClient)
		metricsCollector := metrics.NewCollector(httpClient)
		adapter := edot.NewAdapterWithOptions(cfg, zpagesClient, metricsCollector, edot.AdapterOptions{
			PrometheusURL:  options.edotMetricsURL,
			HealthCheckURL: options.edotHealthURL,
		})
		return adapter.Status(ctx)
	case "otel":
		if strings.TrimSpace(options.otelConfig) == "" {
			return nil, errors.New("otel config not found; pass --otel-config")
		}

		cfg, err := config.ParseOTelCollectorConfig(options.otelConfig)
		if err != nil {
			return nil, err
		}

		httpClient := &http.Client{Timeout: 5 * time.Second}
		zpagesClient := otel.NewZPagesClient(options.otelZPagesURL, httpClient)
		metricsCollector := metrics.NewCollector(httpClient)
		adapter := otel.NewAdapterWithOptions(cfg, zpagesClient, metricsCollector, otel.AdapterOptions{
			PrometheusURL:  options.otelMetricsURL,
			HealthCheckURL: options.otelHealthURL,
		})
		return adapter.Status(ctx)
	default:
		return nil, fmt.Errorf("unsupported --agent value %q", options.agentType)
	}
}

func autoDetectStatusOptions(ctx context.Context, options statusOptions) (statusOptions, error) {
	discovered, err := discoverAgents(ctx)
	if err != nil {
		return options, err
	}
	if len(discovered) == 0 {
		return options, errors.New("no local agents discovered; pass --agent and explicit config flags")
	}

	best := selectPreferredAgent(discovered)
	options.agentType = best.AgentType

	switch best.AgentType {
	case "elastic-agent":
		if strings.TrimSpace(options.elasticConfig) == "" {
			options.elasticConfig = best.ConfigPath
		}
		if endpoint, ok := best.Endpoints["status"]; ok && strings.TrimSpace(options.elasticStatusURL) == "" {
			options.elasticStatusURL = endpoint
		}
	case "edot":
		if strings.TrimSpace(options.edotConfig) == "" {
			options.edotConfig = best.ConfigPath
		}
		if endpoint, ok := best.Endpoints["zpages"]; ok {
			options.edotZPagesURL = endpoint
		}
		if endpoint, ok := best.Endpoints["metrics"]; ok {
			options.edotMetricsURL = endpoint + "/metrics"
		}
		if endpoint, ok := best.Endpoints["health"]; ok {
			options.edotHealthURL = endpoint + "/"
		}
	case "otel":
		if strings.TrimSpace(options.otelConfig) == "" {
			options.otelConfig = best.ConfigPath
		}
		if endpoint, ok := best.Endpoints["zpages"]; ok {
			options.otelZPagesURL = endpoint
		}
		if endpoint, ok := best.Endpoints["metrics"]; ok {
			options.otelMetricsURL = endpoint + "/metrics"
		}
		if endpoint, ok := best.Endpoints["health"]; ok {
			options.otelHealthURL = endpoint + "/"
		}
	}

	return options, nil
}

func selectPreferredAgent(agents []discovery.DiscoveredAgent) discovery.DiscoveredAgent {
	best := agents[0]
	bestScore := discoveryPriority(best.AgentType)
	for i := 1; i < len(agents); i++ {
		score := discoveryPriority(agents[i].AgentType)
		if score < bestScore {
			best = agents[i]
			bestScore = score
		}
	}
	return best
}

func discoveryPriority(agentType string) int {
	switch agentType {
	case "elastic-agent":
		return 0
	case "edot":
		return 1
	case "otel":
		return 2
	default:
		return 3
	}
}

func resolveElasticConfigPath(explicitPath string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}

	for _, candidate := range defaultElasticConfigPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", errors.New("elastic agent config not found; pass --elastic-config")
}

func defaultElasticConfigPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Library/Elastic/Agent/elastic-agent.yml"}
	case "windows":
		return []string{`C:\Program Files\Elastic\Agent\elastic-agent.yml`}
	default:
		return []string{"/opt/Elastic/Agent/elastic-agent.yml"}
	}
}
