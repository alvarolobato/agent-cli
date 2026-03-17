package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestParseElasticAgentConfig_MergesInputsDirYAMLFiles(t *testing.T) {
	dir := t.TempDir()

	mainConfig := []byte(`
outputs:
  default:
    type: elasticsearch
inputs:
  - id: from-main
    type: logfile
    enabled: true
    use_output: default
`)
	if err := os.WriteFile(filepath.Join(dir, "elastic-agent.yml"), mainConfig, 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "inputs.d"), 0o755); err != nil {
		t.Fatalf("Mkdir(inputs.d) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "inputs.d", "20-second.yaml"), []byte(`
inputs:
  - id: from-second
    type: metric
    enabled: true
    use_output: default
`), 0o644); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "inputs.d", "10-first.yml"), []byte(`
inputs:
  - id: from-first
    type: filestream
    enabled: true
    use_output: default
`), 0o644); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "inputs.d", "disabled.yml_no"), []byte(`
inputs:
  - id: should-not-load
    type: filestream
    enabled: true
    use_output: default
`), 0o644); err != nil {
		t.Fatalf("WriteFile(disabled) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "inputs.d", "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("WriteFile(notes) error = %v", err)
	}

	cfg, err := ParseElasticAgentConfig(filepath.Join(dir, "elastic-agent.yml"))
	if err != nil {
		t.Fatalf("ParseElasticAgentConfig() error = %v", err)
	}

	if len(cfg.Inputs) != 3 {
		t.Fatalf("expected 3 inputs after merge, got %d", len(cfg.Inputs))
	}
	if cfg.Inputs[0].ID != "from-main" {
		t.Fatalf("expected first input from main file, got %q", cfg.Inputs[0].ID)
	}
	if cfg.Inputs[1].ID != "from-first" {
		t.Fatalf("expected second input from lexicographically first file, got %q", cfg.Inputs[1].ID)
	}
	if cfg.Inputs[2].ID != "from-second" {
		t.Fatalf("expected third input from lexicographically second file, got %q", cfg.Inputs[2].ID)
	}
}

func TestMergeFromOTelCollectorConfig_MergesSupplementalPipelines(t *testing.T) {
	cfg, err := ParseElasticAgentConfigBytes([]byte(`
outputs:
  default:
    type: elasticsearch
inputs:
  - id: from-main
    type: logfile
    enabled: true
    use_output: default
`))
	if err != nil {
		t.Fatalf("ParseElasticAgentConfigBytes() error = %v", err)
	}

	otelCfg, err := ParseOTelCollectorConfigBytes([]byte(`
exporters:
  elasticsearch/unpoller:
    endpoints: ["https://example.es:443"]
service:
  pipelines:
    metrics/unpoller:
      receivers: ["prometheus/unpoller"]
      exporters: ["elasticsearch/unpoller"]
`))
	if err != nil {
		t.Fatalf("ParseOTelCollectorConfigBytes() error = %v", err)
	}

	MergeFromOTelCollectorConfig(cfg, otelCfg)

	if len(cfg.Inputs) != 2 {
		t.Fatalf("expected 2 inputs after merge, got %d", len(cfg.Inputs))
	}
	if cfg.Inputs[1].ID != "prometheus/unpoller" {
		t.Fatalf("expected synthesized receiver from supplemental otel config, got %q", cfg.Inputs[1].ID)
	}
	if _, ok := cfg.Outputs["elasticsearch/unpoller"]; !ok {
		t.Fatalf("expected synthesized supplemental output to be present")
	}
}
