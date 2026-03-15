package discovery

import (
	"context"

	"github.com/alvarolobato/agent-cli/internal/agent"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

// ProcessInfo describes a running process that might map to an agent.
type ProcessInfo struct {
	PID  int
	Name string
}

// ProcessProvider allows process listing to be mocked in tests.
type ProcessProvider func(ctx context.Context) ([]ProcessInfo, error)

type processScanner struct {
	listProcesses ProcessProvider
}

// NewProcessScanner creates a process-based strategy.
func NewProcessScanner() Strategy {
	return &processScanner{listProcesses: defaultProcessProvider}
}

// NewProcessScannerWithProvider creates a process scanner with a custom process source.
func NewProcessScannerWithProvider(provider ProcessProvider) Strategy {
	if provider == nil {
		provider = defaultProcessProvider
	}
	return &processScanner{listProcesses: provider}
}

func defaultProcessProvider(context.Context) ([]ProcessInfo, error) {
	return []ProcessInfo{}, nil
}

func (s *processScanner) Discover(ctx context.Context) ([]agent.Agent, error) {
	processes, err := s.listProcesses(ctx)
	if err != nil {
		return nil, err
	}

	discovered := make([]agent.Agent, 0, len(processes))
	for _, proc := range processes {
		discovered = append(discovered, stubAgent{
			id:   proc.Name,
			kind: proc.Name,
		})
	}
	return discovered, nil
}

type stubAgent struct {
	id   string
	kind string
}

func (a stubAgent) ID() string { return a.id }

func (a stubAgent) Type() string { return a.kind }

func (a stubAgent) Status(context.Context) (*pipeline.Pipeline, error) {
	return pipeline.ExamplePipeline(), nil
}

func (a stubAgent) Health(context.Context) (pipeline.HealthStatus, error) {
	return pipeline.Unknown, nil
}
