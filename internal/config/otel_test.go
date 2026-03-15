package config

import "testing"

func TestParseOTelCollectorConfig(t *testing.T) {
	cfg, err := ParseOTelCollectorConfig("../../test/fixtures/edot-config.yaml")
	if err != nil {
		t.Fatalf("ParseOTelCollectorConfig() error = %v", err)
	}

	if len(cfg.Receivers) != 2 {
		t.Fatalf("expected 2 receivers, got %d", len(cfg.Receivers))
	}
	if len(cfg.Processors) != 3 {
		t.Fatalf("expected 3 processors, got %d", len(cfg.Processors))
	}
	if len(cfg.Exporters) != 2 {
		t.Fatalf("expected 2 exporters, got %d", len(cfg.Exporters))
	}
	if len(cfg.Extensions) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(cfg.Extensions))
	}
	if len(cfg.Service.Pipelines) != 3 {
		t.Fatalf("expected 3 service pipelines, got %d", len(cfg.Service.Pipelines))
	}

	traces, ok := cfg.Service.Pipelines["traces"]
	if !ok {
		t.Fatalf("expected traces pipeline")
	}
	if traces.Type != OTelPipelineTypeTrace {
		t.Fatalf("expected traces type %q, got %q", OTelPipelineTypeTrace, traces.Type)
	}
	assertStringSlice(t, traces.Receivers, []string{"otlp"})
	assertStringSlice(t, traces.Processors, []string{"memory_limiter", "batch"})
	assertStringSlice(t, traces.Exporters, []string{"debug"})

	metrics, ok := cfg.Service.Pipelines["metrics"]
	if !ok {
		t.Fatalf("expected metrics pipeline")
	}
	if metrics.Type != OTelPipelineTypeMetrics {
		t.Fatalf("expected metrics type %q, got %q", OTelPipelineTypeMetrics, metrics.Type)
	}
	assertStringSlice(t, metrics.Receivers, []string{"otlp", "prometheus/infra"})
	assertStringSlice(t, metrics.Processors, []string{"memory_limiter", "batch"})
	assertStringSlice(t, metrics.Exporters, []string{"debug"})

	logs, ok := cfg.Service.Pipelines["logs"]
	if !ok {
		t.Fatalf("expected logs pipeline")
	}
	if logs.Type != OTelPipelineTypeLogs {
		t.Fatalf("expected logs type %q, got %q", OTelPipelineTypeLogs, logs.Type)
	}
	assertStringSlice(t, logs.Receivers, []string{"otlp"})
	assertStringSlice(t, logs.Processors, []string{"ecsformatprocessor/logs", "batch"})
	assertStringSlice(t, logs.Exporters, []string{"elasticsearch/logs"})
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice length mismatch: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("slice item mismatch at %d: got %q want %q", i, got[i], want[i])
		}
	}
}
