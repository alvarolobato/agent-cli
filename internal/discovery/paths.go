package discovery

import (
	"context"

	"github.com/alvarolobato/agent-cli/internal/agent"
)

type pathScanner struct{}

// NewPathScanner creates a path-based discovery strategy.
func NewPathScanner() Strategy {
	return &pathScanner{}
}

func (s *pathScanner) Discover(context.Context) ([]agent.Agent, error) {
	return []agent.Agent{}, nil
}
