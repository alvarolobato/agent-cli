package config

import "testing"

func TestParseElasticAgentConfig(t *testing.T) {
	cfg, err := ParseElasticAgentConfig("../../test/fixtures/elastic-agent.yml")
	if err != nil {
		t.Fatalf("ParseElasticAgentConfig() error = %v", err)
	}

	if len(cfg.Outputs) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(cfg.Outputs))
	}

	if len(cfg.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(cfg.Inputs))
	}

	systemLogs := cfg.Inputs[0]
	if systemLogs.ID != "system-logs" {
		t.Fatalf("expected first input id system-logs, got %q", systemLogs.ID)
	}
	if !systemLogs.Enabled {
		t.Fatalf("expected system-logs to be enabled")
	}
	if systemLogs.UseOutput != "default" {
		t.Fatalf("expected system-logs use_output default, got %q", systemLogs.UseOutput)
	}
	if len(systemLogs.Streams) != 1 {
		t.Fatalf("expected 1 stream for system-logs, got %d", len(systemLogs.Streams))
	}

	apiEvents := cfg.Inputs[1]
	if apiEvents.ID != "api-events" {
		t.Fatalf("expected second input id api-events, got %q", apiEvents.ID)
	}
	if apiEvents.Enabled {
		t.Fatalf("expected api-events to be disabled")
	}
	if apiEvents.UseOutput != "monitoring" {
		t.Fatalf("expected api-events use_output monitoring, got %q", apiEvents.UseOutput)
	}
}

