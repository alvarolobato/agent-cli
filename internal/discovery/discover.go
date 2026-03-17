package discovery

import (
	"context"
	"fmt"
	"sort"

	"github.com/alvarolobato/agent-cli/internal/agent"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

// Strategy is a single discovery mechanism.
type Strategy interface {
	Discover(ctx context.Context) ([]DiscoveredAgent, error)
}

// Orchestrator runs all discovery strategies and merges the results.
type Orchestrator struct {
	strategies []Strategy
}

// NewOrchestrator builds the default discovery orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		strategies: []Strategy{
			NewProcessScanner(),
			NewServiceScanner(),
			NewInstallDirScanner(),
			NewPathScanner(),
			NewPortProber(),
		},
	}
}

// DiscoveredChild is a process discovered as part of a parent agent runtime.
type DiscoveredChild struct {
	PID  int
	Name string
	Role string
	Args []string
}

// DiscoveredAgent is an enriched discovery result that also satisfies agent.Agent.
type DiscoveredAgent struct {
	AgentType  string
	PID        int
	ConfigPath string
	Endpoints  map[string]string
	Children   []DiscoveredChild
	Source     string
	InstallPath string
	ConfigPaths []string
	Metadata    map[string]string
}

func (a DiscoveredAgent) ID() string {
	agentType := a.AgentType
	if a.PID > 0 {
		return fmt.Sprintf("%s:%d", agentType, a.PID)
	}
	if a.ConfigPath != "" {
		return fmt.Sprintf("%s:%s", agentType, a.ConfigPath)
	}
	if a.InstallPath != "" {
		return fmt.Sprintf("%s:%s", agentType, a.InstallPath)
	}
	return agentType
}

func (a DiscoveredAgent) Type() string {
	return a.AgentType
}

func (a DiscoveredAgent) Status(context.Context) (*pipeline.Pipeline, error) {
	return pipeline.ExamplePipeline(), nil
}

func (a DiscoveredAgent) Health(context.Context) (pipeline.HealthStatus, error) {
	return pipeline.Unknown, nil
}

// DiscoverDetailed executes all strategies and aggregates discovered agents.
func (o *Orchestrator) DiscoverDetailed(ctx context.Context) ([]DiscoveredAgent, error) {
	merged := make([]DiscoveredAgent, 0)

	for _, strategy := range o.strategies {
		agents, err := strategy.Discover(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range agents {
			a = normalizeDiscoveredAgent(a)
			idx := findMergeIndex(merged, a)
			if idx == -1 {
				merged = append(merged, a)
				continue
			}
			merged[idx] = mergeDiscoveredAgent(merged[idx], a)
		}
	}

	out := make([]DiscoveredAgent, 0, len(merged))
	for _, a := range merged {
		out = append(out, finalizeDiscoveredAgent(a))
	}
	return out, nil
}

// Discover executes all strategies and aggregates discovered agents.
func (o *Orchestrator) Discover(ctx context.Context) ([]agent.Agent, error) {
	detailed, err := o.DiscoverDetailed(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]agent.Agent, 0, len(detailed))
	for _, a := range detailed {
		out = append(out, a)
	}
	return out, nil
}

func findMergeIndex(merged []DiscoveredAgent, incoming DiscoveredAgent) int {
	if incoming.PID > 0 {
		for i := range merged {
			if merged[i].AgentType == incoming.AgentType && merged[i].PID == incoming.PID {
				return i
			}
		}
	}
	if incoming.ConfigPath != "" {
		for i := range merged {
			if merged[i].AgentType == incoming.AgentType && merged[i].ConfigPath == incoming.ConfigPath && merged[i].ConfigPath != "" {
				return i
			}
		}
	}
	if idx := findByEndpointOverlap(merged, incoming); idx != -1 {
		return idx
	}
	typeMatches := make([]int, 0, len(merged))
	for i := range merged {
		if merged[i].AgentType == incoming.AgentType {
			typeMatches = append(typeMatches, i)
		}
	}
	if len(typeMatches) != 1 {
		return -1
	}
	only := typeMatches[0]
	if canFallbackMerge(merged[only], incoming) {
		return only
	}
	return -1
}

func findByEndpointOverlap(merged []DiscoveredAgent, incoming DiscoveredAgent) int {
	if len(incoming.Endpoints) == 0 {
		return -1
	}
	for i := range merged {
		if merged[i].AgentType != incoming.AgentType || len(merged[i].Endpoints) == 0 {
			continue
		}
		for key, value := range incoming.Endpoints {
			if value == "" {
				continue
			}
			if merged[i].Endpoints[key] == value {
				return i
			}
		}
	}
	return -1
}

func canFallbackMerge(existing DiscoveredAgent, incoming DiscoveredAgent) bool {
	if existing.PID > 0 && incoming.PID > 0 && existing.PID != incoming.PID {
		return false
	}
	if existing.ConfigPath != "" &&
		incoming.ConfigPath != "" &&
		existing.ConfigPath != incoming.ConfigPath &&
		existing.Source == "path" &&
		incoming.Source == "path" {
		return false
	}
	return true
}

func normalizeDiscoveredAgent(a DiscoveredAgent) DiscoveredAgent {
	if a.Endpoints == nil {
		a.Endpoints = map[string]string{}
	}
	if a.Children == nil {
		a.Children = []DiscoveredChild{}
	}
	return a
}

func mergeDiscoveredAgent(existing DiscoveredAgent, incoming DiscoveredAgent) DiscoveredAgent {
	existing = normalizeDiscoveredAgent(existing)
	incoming = normalizeDiscoveredAgent(incoming)

	if existing.PID == 0 {
		existing.PID = incoming.PID
	}
	if shouldPreferConfigPath(existing, incoming) {
		existing.ConfigPath = incoming.ConfigPath
	}
	if sourcePriority(incoming.Source) < sourcePriority(existing.Source) {
		existing.Source = incoming.Source
	}
	for k, v := range incoming.Endpoints {
		if _, ok := existing.Endpoints[k]; !ok && v != "" {
			existing.Endpoints[k] = v
		}
	}
	existing.Children = append(existing.Children, incoming.Children...)
	return existing
}

func shouldPreferConfigPath(existing DiscoveredAgent, incoming DiscoveredAgent) bool {
	incomingPath := incoming.ConfigPath
	existingPath := existing.ConfigPath
	if incomingPath == "" {
		return false
	}
	if existingPath == "" {
		return true
	}
	return configSourcePriority(incoming.Source) < configSourcePriority(existing.Source)
}

func finalizeDiscoveredAgent(a DiscoveredAgent) DiscoveredAgent {
	a = normalizeDiscoveredAgent(a)
	sort.SliceStable(a.Children, func(i, j int) bool {
		if a.Children[i].PID != a.Children[j].PID {
			return a.Children[i].PID < a.Children[j].PID
		}
		return a.Children[i].Role < a.Children[j].Role
	})
	return a
}

func sourcePriority(source string) int {
	switch source {
	case "process":
		return 0
	case "path":
		return 1
	case "port":
		return 2
	default:
		return 3
	}
}

func configSourcePriority(source string) int {
	switch source {
	case "path":
		return 0
	case "process":
		return 1
	case "port":
		return 2
	default:
		return 3
	}
}
