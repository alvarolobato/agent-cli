package elasticagent

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
	"github.com/alvarolobato/agent-cli/test/mocks"
)

func TestAdapterWithMockElasticAgentHandler(t *testing.T) {
	cfg, err := config.ParseElasticAgentConfig("../../../test/fixtures/elastic-agent.yml")
	if err != nil {
		t.Fatalf("ParseElasticAgentConfig() error = %v", err)
	}

	server := httptest.NewServer(mocks.ElasticAgentHandler())
	defer server.Close()

	adapter := NewAdapter(cfg, NewClient(server.URL, server.Client()))
	status, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.Name == "" {
		t.Fatalf("expected non-empty pipeline name")
	}
	if len(status.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(status.Nodes))
	}

	health, err := adapter.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health != pipeline.Healthy {
		t.Fatalf("expected health healthy, got %q", health)
	}
}
