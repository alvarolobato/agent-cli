package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/metrics"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

const (
	defaultAgentID          = "otel-collector"
	defaultAgentType        = "otel"
	defaultPipelineName     = "otel"
	defaultPrometheusURL    = "http://localhost:8888/metrics"
	defaultHealthCheckURL   = "http://localhost:13133/"
	defaultTracezReachable  = "tracez_reachable"
	defaultAgentTypeKey     = "agent_type"
	defaultUnknownComponent = "unknown"
)

type zpagesAPI interface {
	GetPipelineTopology(ctx context.Context) (*PipelineTopology, error)
}

type otelMetricsAPI interface {
	CollectOTelPrometheus(ctx context.Context, endpoint string) (*metrics.OTelSnapshot, error)
}

// NodeLabeler customizes labels rendered for pipeline components.
type NodeLabeler func(componentName, kind, pipelineName string, descriptor ComponentDescriptor) string

// MetadataAugmenter adds extra metadata fields based on built nodes.
type MetadataAugmenter func(nodes []pipeline.Node) map[string]string

// Adapter maps OTel collector config and runtime APIs into the shared pipeline model.
type Adapter struct {
	cfg               *config.OTelCollectorConfig
	zpages            zpagesAPI
	metrics           otelMetricsAPI
	httpClient        *http.Client
	prometheusURL     string
	healthCheckURL    string
	agentID           string
	agentType         string
	pipelineName      string
	nodeLabeler       NodeLabeler
	metadataAugmenter MetadataAugmenter
}

// AdapterOptions overrides default runtime endpoints and optional labeling behavior.
type AdapterOptions struct {
	PrometheusURL     string
	HealthCheckURL    string
	HTTPClient        *http.Client
	AgentID           string
	AgentType         string
	PipelineName      string
	NodeLabeler       NodeLabeler
	MetadataAugmenter MetadataAugmenter
}

// NewAdapter creates a generic OTel adapter with defaults for local collector endpoints.
func NewAdapter(cfg *config.OTelCollectorConfig, zpagesClient *ZPagesClient, metricsCollector *metrics.Collector) *Adapter {
	return NewAdapterWithOptions(cfg, zpagesClient, metricsCollector, AdapterOptions{})
}

// NewAdapterWithOptions creates an OTel adapter with configurable endpoints and metadata behavior.
func NewAdapterWithOptions(
	cfg *config.OTelCollectorConfig,
	zpagesClient zpagesAPI,
	metricsCollector otelMetricsAPI,
	options AdapterOptions,
) *Adapter {
	return newAdapterWithDeps(cfg, zpagesClient, metricsCollector, options.HTTPClient, options)
}

func newAdapterWithDeps(
	cfg *config.OTelCollectorConfig,
	zpagesClient zpagesAPI,
	metricsCollector otelMetricsAPI,
	httpClient *http.Client,
	options AdapterOptions,
) *Adapter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	if strings.TrimSpace(options.PrometheusURL) == "" {
		options.PrometheusURL = defaultPrometheusURL
	}
	if strings.TrimSpace(options.HealthCheckURL) == "" {
		options.HealthCheckURL = defaultHealthCheckURL
	}
	if strings.TrimSpace(options.AgentID) == "" {
		options.AgentID = defaultAgentID
	}
	if strings.TrimSpace(options.AgentType) == "" {
		options.AgentType = defaultAgentType
	}
	if strings.TrimSpace(options.PipelineName) == "" {
		options.PipelineName = defaultPipelineName
	}
	if options.NodeLabeler == nil {
		options.NodeLabeler = defaultNodeLabel
	}

	return &Adapter{
		cfg:               cfg,
		zpages:            zpagesClient,
		metrics:           metricsCollector,
		httpClient:        httpClient,
		prometheusURL:     options.PrometheusURL,
		healthCheckURL:    options.HealthCheckURL,
		agentID:           options.AgentID,
		agentType:         options.AgentType,
		pipelineName:      options.PipelineName,
		nodeLabeler:       options.NodeLabeler,
		metadataAugmenter: options.MetadataAugmenter,
	}
}

// ID returns the adapter identifier.
func (a *Adapter) ID() string {
	return a.agentID
}

// Type returns the adapter type string.
func (a *Adapter) Type() string {
	return a.agentType
}

// Status builds a pipeline graph from config wiring and runtime status/metrics.
func (a *Adapter) Status(ctx context.Context) (*pipeline.Pipeline, error) {
	if a.cfg == nil {
		return nil, fmt.Errorf("otel collector config is required")
	}
	if a.zpages == nil {
		return nil, fmt.Errorf("zpages client is required")
	}

	topology, err := a.zpages.GetPipelineTopology(ctx)
	if err != nil {
		return nil, err
	}

	var metricSnapshot *metrics.OTelSnapshot
	if a.metrics != nil {
		metricSnapshot, _ = a.metrics.CollectOTelPrometheus(ctx, a.prometheusURL)
	}

	statusByKey := flattenTopologyStatus(topology)

	nodes := make([]pipeline.Node, 0)
	edges := make([]pipeline.Edge, 0)
	nodeByID := map[string]int{}

	pipelineNames := make([]string, 0, len(a.cfg.Service.Pipelines))
	for name := range a.cfg.Service.Pipelines {
		pipelineNames = append(pipelineNames, name)
	}
	sort.Strings(pipelineNames)

	for _, pipelineName := range pipelineNames {
		p := a.cfg.Service.Pipelines[pipelineName]
		receivers := copied(p.Receivers)
		processors := copied(p.Processors)
		exporters := copied(p.Exporters)

		firstProcessOrExport := ""
		if len(processors) > 0 {
			firstProcessOrExport = processors[0]
		} else if len(exporters) > 0 {
			firstProcessOrExport = exporters[0]
		}

		for _, receiver := range receivers {
			id := "receiver." + receiver
			node := a.buildNode(receiver, "receiver", pipelineName, statusByKey, metricSnapshot)
			upsertNode(&nodes, nodeByID, id, node)
			if firstProcessOrExport != "" {
				edges = append(edges, pipeline.Edge{
					From: id,
					To:   kindPrefix(firstProcessOrExport, processors, "processor", "exporter"),
				})
			}
		}

		for i, processor := range processors {
			id := "processor." + processor
			node := a.buildNode(processor, "processor", pipelineName, statusByKey, metricSnapshot)
			upsertNode(&nodes, nodeByID, id, node)

			if i+1 < len(processors) {
				edges = append(edges, pipeline.Edge{
					From: id,
					To:   "processor." + processors[i+1],
				})
				continue
			}
			for _, exporter := range exporters {
				edges = append(edges, pipeline.Edge{
					From: id,
					To:   "exporter." + exporter,
				})
			}
		}

		for _, exporter := range exporters {
			id := "exporter." + exporter
			node := a.buildNode(exporter, "exporter", pipelineName, statusByKey, metricSnapshot)
			upsertNode(&nodes, nodeByID, id, node)
		}
	}

	metadata := map[string]string{
		defaultAgentTypeKey:    a.agentType,
		defaultTracezReachable: fmt.Sprintf("%t", topology != nil && topology.TracezReachable),
	}
	for key, value := range a.extraMetadata(nodes) {
		metadata[key] = value
	}

	return &pipeline.Pipeline{
		Name:      a.pipelineName,
		Nodes:     nodes,
		Edges:     dedupeEdges(edges),
		UpdatedAt: time.Now().UTC(),
		Metadata:  metadata,
	}, nil
}

// Health returns top-level health using the health_check endpoint.
func (a *Adapter) Health(ctx context.Context) (pipeline.HealthStatus, error) {
	url := healthURL(a.cfg, a.healthCheckURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return pipeline.Unknown, fmt.Errorf("build health check request: %w", err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return pipeline.Unknown, fmt.Errorf("request health check endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return pipeline.Error, nil
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return pipeline.Unknown, fmt.Errorf("decode health check response: %w", err)
	}
	status := strings.TrimSpace(readString(payload, "status"))
	switch strings.ToLower(status) {
	case "server available", "statusok", "ok":
		return pipeline.Healthy, nil
	case "server not available", "statusfatalerror", "statuspermanenterror":
		return pipeline.Error, nil
	default:
		return pipeline.Unknown, nil
	}
}

func (a *Adapter) buildNode(
	componentName string,
	kind string,
	pipelineName string,
	statusByKey map[string]string,
	snapshot *metrics.OTelSnapshot,
) pipeline.Node {
	id := kind + "." + componentName
	rawStatus := statusByKey[pipelineStatusKey(pipelineName, kind, componentName)]
	status := pipeline.MapOTelRuntimeStatus(rawStatus)
	descriptor := LookupComponentDescriptor(kind, componentName, pipelineName)
	label := a.nodeLabeler(componentName, kind, pipelineName, descriptor)
	if strings.TrimSpace(label) == "" {
		label = defaultUnknownComponent
	}

	node := pipeline.Node{
		ID:     id,
		Label:  label,
		Kind:   kind,
		Status: status,
	}

	if snapshot == nil {
		return node
	}

	switch kind {
	case "receiver":
		component := snapshot.Receivers[componentName]
		node.Metrics = &pipeline.NodeMetrics{
			EventsInPerSec: component.Accepted,
		}
		node.Status = pipeline.AssessOTelComponentHealth(node.Status, component.SendFailed, component.Dropped, true)
	case "processor":
		component := snapshot.Processors[componentName]
		node.Metrics = &pipeline.NodeMetrics{
			DropCount: component.Dropped,
		}
		node.Status = pipeline.AssessOTelComponentHealth(node.Status, component.SendFailed, component.Dropped, true)
	case "exporter":
		component := snapshot.Exporters[componentName]
		node.Metrics = &pipeline.NodeMetrics{
			EventsOutPerSec: component.Sent,
			ErrorCount:      component.SendFailed,
		}
		node.Status = pipeline.AssessOTelComponentHealth(node.Status, component.SendFailed, component.Dropped, true)
	}

	return node
}

func defaultNodeLabel(componentName, _, _ string, descriptor ComponentDescriptor) string {
	if strings.TrimSpace(descriptor.Name) != "" {
		return descriptor.Name
	}
	return componentName
}

func (a *Adapter) extraMetadata(nodes []pipeline.Node) map[string]string {
	if a.metadataAugmenter == nil {
		return nil
	}
	return a.metadataAugmenter(nodes)
}

func flattenTopologyStatus(topology *PipelineTopology) map[string]string {
	out := map[string]string{}
	if topology == nil {
		return out
	}
	for _, p := range topology.Pipelines {
		for _, receiver := range p.Receivers {
			out[pipelineStatusKey(p.Name, "receiver", receiver.ID)] = receiver.Status
		}
		for _, processor := range p.Processors {
			out[pipelineStatusKey(p.Name, "processor", processor.ID)] = processor.Status
		}
		for _, exporter := range p.Exporters {
			out[pipelineStatusKey(p.Name, "exporter", exporter.ID)] = exporter.Status
		}
	}
	return out
}

func pipelineStatusKey(pipelineName, kind, componentName string) string {
	return strings.ToLower(strings.TrimSpace(pipelineName + "|" + kind + "|" + componentName))
}

func upsertNode(nodes *[]pipeline.Node, nodeByID map[string]int, id string, node pipeline.Node) {
	current := *nodes
	if idx, ok := nodeByID[id]; ok {
		if healthRank(node.Status) > healthRank(current[idx].Status) {
			current[idx].Status = node.Status
		}
		*nodes = current
		return
	}
	nodeByID[id] = len(current)
	*nodes = append(current, node)
}

func kindPrefix(name string, processors []string, processorPrefix, exporterPrefix string) string {
	for _, processor := range processors {
		if processor == name {
			return processorPrefix + "." + name
		}
	}
	return exporterPrefix + "." + name
}

func healthRank(status pipeline.HealthStatus) int {
	switch status {
	case pipeline.Error:
		return 4
	case pipeline.Degraded:
		return 3
	case pipeline.Disabled:
		return 2
	case pipeline.Unknown:
		return 1
	default:
		return 0
	}
}

func dedupeEdges(edges []pipeline.Edge) []pipeline.Edge {
	seen := map[string]bool{}
	out := make([]pipeline.Edge, 0, len(edges))
	for _, edge := range edges {
		key := edge.From + "->" + edge.To
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, edge)
	}
	return out
}

func copied(values []string) []string {
	out := append([]string(nil), values...)
	return out
}

func readString(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, _ := payload[key].(string)
	return value
}

func healthURL(cfg *config.OTelCollectorConfig, fallback string) string {
	if cfg == nil {
		return fallback
	}
	component, ok := cfg.Extensions["health_check"]
	if !ok {
		return fallback
	}
	rawEndpoint, ok := component.Raw["endpoint"]
	if !ok {
		return fallback
	}
	endpoint, ok := rawEndpoint.(string)
	if !ok {
		return fallback
	}
	value := strings.TrimSpace(endpoint)
	if value == "" {
		return fallback
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return strings.TrimRight(value, "/") + "/"
	}
	return "http://" + strings.TrimRight(value, "/") + "/"
}
