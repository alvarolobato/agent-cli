package elasticagent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/metrics"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

const (
	agentID   = "elastic-agent"
	agentType = "elastic-agent"
)

type statusAPI interface {
	GetStatus(ctx context.Context) (*StatusResponse, error)
}

type beatMetricsAPI interface {
	CollectBeatStats(ctx context.Context, endpoint string) (*metrics.Snapshot, error)
}

// Adapter maps Elastic Agent config + runtime status into the shared pipeline model.
type Adapter struct {
	cfg                    *config.ElasticAgentConfig
	client                 statusAPI
	metrics                beatMetricsAPI
	metricsEndpointByInput func(config.ElasticInput) string
}

// NewAdapter creates an adapter with an explicit config and status client.
func NewAdapter(cfg *config.ElasticAgentConfig, client *Client) *Adapter {
	return newAdapterWithDeps(cfg, client, metrics.NewCollector(nil), defaultMetricsEndpointByInput)
}

func newAdapterWithDeps(
	cfg *config.ElasticAgentConfig,
	client statusAPI,
	metricsCollector beatMetricsAPI,
	metricsEndpointByInput func(config.ElasticInput) string,
) *Adapter {
	return &Adapter{
		cfg:                    cfg,
		client:                 client,
		metrics:                metricsCollector,
		metricsEndpointByInput: metricsEndpointByInput,
	}
}

// ID returns the adapter identifier.
func (a *Adapter) ID() string {
	return agentID
}

// Type returns the adapter type string.
func (a *Adapter) Type() string {
	return agentType
}

// Status builds a pipeline from elastic-agent.yml wiring and runtime status payload.
func (a *Adapter) Status(ctx context.Context) (*pipeline.Pipeline, error) {
	if a.cfg == nil {
		return nil, fmt.Errorf("elastic agent config is required")
	}
	if a.client == nil {
		return nil, fmt.Errorf("elastic agent status client is required")
	}

	runtime, err := a.client.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	components := indexComponents(runtime.Components)
	nodes := make([]pipeline.Node, 0, len(a.cfg.Inputs)+len(a.cfg.Outputs))
	edges := make([]pipeline.Edge, 0, len(a.cfg.Inputs))
	metricsByEndpoint := map[string]*pipeline.NodeMetrics{}

	outputNodeIDs := make(map[string]string, len(a.cfg.Outputs))
	outputNames := make([]string, 0, len(a.cfg.Outputs))
	for outputName := range a.cfg.Outputs {
		outputNames = append(outputNames, outputName)
	}
	sort.Strings(outputNames)

	for _, outputName := range outputNames {
		status := resolveOutputStatus(outputName, components)
		nodeID := fmt.Sprintf("output.%s", outputName)
		outputNodeIDs[outputName] = nodeID
		nodes = append(nodes, pipeline.Node{
			ID:     nodeID,
			Label:  outputName,
			Kind:   "output",
			Status: status,
		})
	}

	for _, in := range a.cfg.Inputs {
		status := resolveInputStatus(in, components)
		if !in.Enabled {
			status = pipeline.Disabled
		}

		nodeID := fmt.Sprintf("input.%s", in.ID)
		nodes = append(nodes, pipeline.Node{
			ID:      nodeID,
			Label:   in.ID,
			Kind:    "input",
			Status:  status,
			Metrics: a.resolveInputMetrics(ctx, in, metricsByEndpoint),
		})

		if outputNodeID, ok := outputNodeIDs[in.UseOutput]; ok {
			edges = append(edges, pipeline.Edge{
				From: nodeID,
				To:   outputNodeID,
			})
		}
	}

	return &pipeline.Pipeline{
		Name:      runtime.Name,
		Nodes:     nodes,
		Edges:     edges,
		UpdatedAt: time.Now().UTC(),
		Metadata: map[string]string{
			"agent_id":       runtime.ID,
			"agent_type":     agentType,
			"runtime_status": strings.ToLower(runtime.Status.Overall),
		},
	}, nil
}

func (a *Adapter) resolveInputMetrics(ctx context.Context, in config.ElasticInput, cache map[string]*pipeline.NodeMetrics) *pipeline.NodeMetrics {
	if !in.Enabled || a.metrics == nil || a.metricsEndpointByInput == nil {
		return nil
	}

	endpoint := strings.TrimSpace(a.metricsEndpointByInput(in))
	if endpoint == "" {
		return nil
	}

	if cached, ok := cache[endpoint]; ok {
		return cached
	}

	snapshot, err := a.metrics.CollectBeatStats(ctx, endpoint)
	if err != nil || snapshot == nil {
		cache[endpoint] = nil
		return nil
	}

	nodeMetrics := &pipeline.NodeMetrics{
		EventsInPerSec:  snapshot.EventsInPerSec,
		EventsOutPerSec: snapshot.EventsOutPerSec,
		ErrorCount:      snapshot.ErrorCount,
		DropCount:       snapshot.DropCount,
	}
	cache[endpoint] = nodeMetrics
	return nodeMetrics
}

// Health returns just the top-level runtime health.
func (a *Adapter) Health(ctx context.Context) (pipeline.HealthStatus, error) {
	if a.client == nil {
		return pipeline.Unknown, fmt.Errorf("elastic agent status client is required")
	}
	runtime, err := a.client.GetStatus(ctx)
	if err != nil {
		return pipeline.Unknown, err
	}
	return mapRuntimeStatus(runtime.Status.Overall), nil
}

type componentIndex struct {
	byID   map[string]pipeline.HealthStatus
	byName map[string]pipeline.HealthStatus
}

func indexComponents(components []ComponentInfo) componentIndex {
	idx := componentIndex{
		byID:   make(map[string]pipeline.HealthStatus, len(components)),
		byName: make(map[string]pipeline.HealthStatus, len(components)),
	}

	for _, component := range components {
		status := mapRuntimeStatus(component.Status.Overall)
		id := strings.ToLower(strings.TrimSpace(component.ID))
		name := strings.ToLower(strings.TrimSpace(component.Name))
		if id != "" {
			idx.byID[id] = status
		}
		if name != "" {
			idx.byName[name] = status
		}
	}

	return idx
}

func resolveInputStatus(in config.ElasticInput, components componentIndex) pipeline.HealthStatus {
	candidates := []string{
		fmt.Sprintf("%s-%s", in.Type, in.UseOutput),
		fmt.Sprintf("%s/%s", in.Type, in.UseOutput),
		in.Type,
		in.ID,
		fmt.Sprintf("%s-%s", in.ID, in.UseOutput),
	}
	return resolveByCandidates(candidates, components)
}

func resolveOutputStatus(outputName string, components componentIndex) pipeline.HealthStatus {
	candidates := []string{
		outputName,
		fmt.Sprintf("output-%s", outputName),
		fmt.Sprintf("output.%s", outputName),
		fmt.Sprintf("es-%s", outputName),
	}
	return resolveByCandidates(candidates, components)
}

func resolveByCandidates(candidates []string, components componentIndex) pipeline.HealthStatus {
	for _, candidate := range candidates {
		key := strings.ToLower(strings.TrimSpace(candidate))
		if key == "" {
			continue
		}
		if status, ok := components.byID[key]; ok {
			return status
		}
		if status, ok := components.byName[key]; ok {
			return status
		}
	}
	return pipeline.Unknown
}

func mapRuntimeStatus(overall string) pipeline.HealthStatus {
	switch strings.ToUpper(strings.TrimSpace(overall)) {
	case "HEALTHY", "RUNNING", "OK":
		return pipeline.Healthy
	case "DEGRADED", "WARNING":
		return pipeline.Degraded
	case "FAILED", "ERROR":
		return pipeline.Error
	case "DISABLED", "STOPPED":
		return pipeline.Disabled
	default:
		return pipeline.Unknown
	}
}

func defaultMetricsEndpointByInput(_ config.ElasticInput) string {
	return "http://localhost:5066/stats"
}
