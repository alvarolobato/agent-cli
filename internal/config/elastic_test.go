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

func TestParseElasticAgentConfigBytes_OTELPipelineShape(t *testing.T) {
	raw := []byte(`
outputs:
  default:
    type: elasticsearch
    hosts: ["https://example.es:443"]

receivers:
  prometheus/unpoller:
    config:
      scrape_configs:
        - job_name: "unpoller"
exporters:
  elasticsearch/unpoller:
    endpoints: ["https://example.es:443"]
service:
  pipelines:
    metrics/unpoller:
      receivers: ["prometheus/unpoller"]
      processors: ["resource/unpoller"]
      exporters: ["elasticsearch/unpoller"]
`)

	cfg, err := ParseElasticAgentConfigBytes(raw)
	if err != nil {
		t.Fatalf("ParseElasticAgentConfigBytes() error = %v", err)
	}

	if len(cfg.Inputs) != 1 {
		t.Fatalf("expected 1 synthesized input, got %d", len(cfg.Inputs))
	}

	derived := cfg.Inputs[0]
	if derived.ID != "prometheus/unpoller" {
		t.Fatalf("expected receiver id prometheus/unpoller, got %q", derived.ID)
	}
	if derived.Type != "prometheus" {
		t.Fatalf("expected receiver type prometheus, got %q", derived.Type)
	}
	if !derived.Enabled {
		t.Fatalf("expected synthesized input to be enabled")
	}
	if derived.UseOutput != "elasticsearch/unpoller" {
		t.Fatalf("expected synthesized input use_output elasticsearch/unpoller, got %q", derived.UseOutput)
	}

	exp, ok := cfg.Outputs["elasticsearch/unpoller"]
	if !ok {
		t.Fatalf("expected synthesized output elasticsearch/unpoller")
	}
	if exp.Type != "elasticsearch" {
		t.Fatalf("expected synthesized output type elasticsearch, got %q", exp.Type)
	}
	if len(exp.Hosts) != 1 || exp.Hosts[0] != "https://example.es:443" {
		t.Fatalf("expected synthesized output hosts to match endpoint, got %#v", exp.Hosts)
	}
}
