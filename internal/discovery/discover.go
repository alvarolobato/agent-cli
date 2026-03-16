package discovery

import (
	"context"
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
}

func (a DiscoveredAgent) ID() string {
	if a.PID > 0 {
		return a.AgentType
	}
	if a.ConfigPath != "" {
		return a.ConfigPath
	}
	return a.AgentType
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
	merged := make(map[string]DiscoveredAgent)
	order := make([]string, 0)

	for _, strategy := range o.strategies {
		agents, err := strategy.Discover(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range agents {
			key := discoveredAgentKey(a)
			if _, ok := merged[key]; !ok {
				merged[key] = normalizeDiscoveredAgent(a)
				order = append(order, key)
				continue
			}
			existing := merged[key]
			merged[key] = mergeDiscoveredAgent(existing, a)
		}
	}

	out := make([]DiscoveredAgent, 0, len(order))
	for _, key := range order {
		out = append(out, finalizeDiscoveredAgent(merged[key]))
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

func discoveredAgentKey(a DiscoveredAgent) string {
	return a.AgentType
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
