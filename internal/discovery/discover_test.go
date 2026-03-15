package discovery

import (
	"context"
	"testing"
)

func TestProcessScannerWithMockedProcessList(t *testing.T) {
	mockProvider := func(context.Context) ([]ProcessInfo, error) {
		return []ProcessInfo{
			{PID: 100, Name: "elastic-agent"},
			{PID: 200, Name: "otelcol"},
		}, nil
	}

	strategy := NewProcessScannerWithProvider(mockProvider)
	agents, err := strategy.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].ID() != "elastic-agent" {
		t.Fatalf("expected first agent elastic-agent, got %q", agents[0].ID())
	}
}
