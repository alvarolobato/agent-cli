package discovery

import (
	"context"

	"github.com/alvarolobato/agent-cli/internal/agent"
)

// Strategy is a single discovery mechanism.
type Strategy interface {
	Discover(ctx context.Context) ([]agent.Agent, error)
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

// Discover executes all strategies and aggregates discovered agents.
func (o *Orchestrator) Discover(ctx context.Context) ([]agent.Agent, error) {
	out := make([]agent.Agent, 0)
	for _, strategy := range o.strategies {
		agents, err := strategy.Discover(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, agents...)
	}
	return out, nil
}
