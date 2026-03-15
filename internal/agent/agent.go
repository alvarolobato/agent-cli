package agent

import (
	"context"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

// Agent describes the minimum contract for any supported runtime agent.
type Agent interface {
	ID() string
	Type() string
	Status(ctx context.Context) (*pipeline.Pipeline, error)
	Health(ctx context.Context) (pipeline.HealthStatus, error)
}
