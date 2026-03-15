package discovery

import (
	"context"

	"github.com/alvarolobato/agent-cli/internal/agent"
)

type portProber struct{}

// NewPortProber creates a port-based discovery strategy.
func NewPortProber() Strategy {
	return &portProber{}
}

func (s *portProber) Discover(context.Context) ([]agent.Agent, error) {
	return []agent.Agent{}, nil
}
