package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// OTelCollectorConfig models OTel/EDOT collector YAML fields needed by the adapter.
type OTelCollectorConfig struct {
	Receivers  map[string]OTelComponent `yaml:"receivers"`
	Processors map[string]OTelComponent `yaml:"processors"`
	Exporters  map[string]OTelComponent `yaml:"exporters"`
	Extensions map[string]OTelComponent `yaml:"extensions"`
	Service    OTelServiceConfig        `yaml:"service"`
}

// OTelComponent keeps arbitrary component config while exposing typed metadata.
type OTelComponent struct {
	Name string
	Type string
	Raw  map[string]interface{}
}

// OTelServiceConfig wraps configured collector pipelines.
type OTelServiceConfig struct {
	Pipelines map[string]OTelPipelineConfig `yaml:"pipelines"`
}

// OTelPipelineConfig is one service pipeline wiring definition.
type OTelPipelineConfig struct {
	Name       string
	Type       OTelPipelineType
	Receivers  []string `yaml:"receivers"`
	Processors []string `yaml:"processors"`
	Exporters  []string `yaml:"exporters"`
}

// OTelPipelineType identifies signal type for a pipeline.
type OTelPipelineType string

const (
	OTelPipelineTypeTrace   OTelPipelineType = "trace"
	OTelPipelineTypeMetrics OTelPipelineType = "metrics"
	OTelPipelineTypeLogs    OTelPipelineType = "logs"
	OTelPipelineTypeUnknown OTelPipelineType = "unknown"
)

type rawOTelCollectorConfig struct {
	Receivers  map[string]map[string]interface{} `yaml:"receivers"`
	Processors map[string]map[string]interface{} `yaml:"processors"`
	Exporters  map[string]map[string]interface{} `yaml:"exporters"`
	Extensions map[string]map[string]interface{} `yaml:"extensions"`
	Service    struct {
		Pipelines map[string]OTelPipelineConfig `yaml:"pipelines"`
	} `yaml:"service"`
}

// ParseOTelCollectorConfig reads and parses OTel/EDOT collector YAML from disk.
func ParseOTelCollectorConfig(path string) (*OTelCollectorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read otel collector config: %w", err)
	}
	return ParseOTelCollectorConfigBytes(data)
}

// ParseOTelCollectorConfigBytes parses raw YAML bytes into OTelCollectorConfig.
func ParseOTelCollectorConfigBytes(data []byte) (*OTelCollectorConfig, error) {
	var raw rawOTelCollectorConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse otel collector config yaml: %w", err)
	}

	cfg := &OTelCollectorConfig{
		Receivers:  makeComponents(raw.Receivers),
		Processors: makeComponents(raw.Processors),
		Exporters:  makeComponents(raw.Exporters),
		Extensions: makeComponents(raw.Extensions),
		Service: OTelServiceConfig{
			Pipelines: map[string]OTelPipelineConfig{},
		},
	}

	for name, pipeline := range raw.Service.Pipelines {
		pipeline.Name = name
		pipeline.Type = pipelineTypeFromName(name)
		cfg.Service.Pipelines[name] = pipeline
	}

	return cfg, nil
}

func makeComponents(source map[string]map[string]interface{}) map[string]OTelComponent {
	dest := map[string]OTelComponent{}
	for name, raw := range source {
		dest[name] = OTelComponent{
			Name: name,
			Type: segmentBeforeSlash(name),
			Raw:  raw,
		}
	}
	return dest
}

func pipelineTypeFromName(name string) OTelPipelineType {
	base := segmentBeforeSlash(name)
	switch base {
	case "trace", "traces":
		return OTelPipelineTypeTrace
	case "metrics":
		return OTelPipelineTypeMetrics
	case "logs":
		return OTelPipelineTypeLogs
	default:
		return OTelPipelineTypeUnknown
	}
}

func segmentBeforeSlash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	idx := strings.Index(trimmed, "/")
	if idx <= 0 {
		return trimmed
	}
	return trimmed[:idx]
}
