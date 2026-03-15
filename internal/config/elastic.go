package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ElasticAgentConfig models the subset of elastic-agent.yml needed by Phase 1A.
type ElasticAgentConfig struct {
	Outputs map[string]ElasticOutput `yaml:"outputs"`
	Inputs  []ElasticInput           `yaml:"inputs"`
}

// ElasticOutput is a named output target in elastic-agent.yml.
type ElasticOutput struct {
	Type  string   `yaml:"type"`
	Hosts []string `yaml:"hosts,omitempty"`
}

// ElasticInput represents a single Elastic Agent input block.
type ElasticInput struct {
	ID        string                   `yaml:"id"`
	Type      string                   `yaml:"type"`
	Enabled   bool                     `yaml:"enabled"`
	UseOutput string                   `yaml:"use_output"`
	Streams   []map[string]interface{} `yaml:"streams,omitempty"`
}

// UnmarshalYAML defaults enabled to true when the field is omitted.
func (in *ElasticInput) UnmarshalYAML(value *yaml.Node) error {
	type rawInput struct {
		ID        string                   `yaml:"id"`
		Type      string                   `yaml:"type"`
		Enabled   *bool                    `yaml:"enabled"`
		UseOutput string                   `yaml:"use_output"`
		Streams   []map[string]interface{} `yaml:"streams,omitempty"`
	}

	var raw rawInput
	if err := value.Decode(&raw); err != nil {
		return err
	}

	in.ID = raw.ID
	in.Type = raw.Type
	in.UseOutput = raw.UseOutput
	in.Streams = raw.Streams
	in.Enabled = true
	if raw.Enabled != nil {
		in.Enabled = *raw.Enabled
	}
	return nil
}

// ParseElasticAgentConfig reads and parses elastic-agent.yml from disk.
func ParseElasticAgentConfig(path string) (*ElasticAgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read elastic agent config: %w", err)
	}
	return ParseElasticAgentConfigBytes(data)
}

// ParseElasticAgentConfigBytes parses raw YAML bytes into ElasticAgentConfig.
func ParseElasticAgentConfigBytes(data []byte) (*ElasticAgentConfig, error) {
	var cfg ElasticAgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse elastic agent config yaml: %w", err)
	}

	if cfg.Outputs == nil {
		cfg.Outputs = map[string]ElasticOutput{}
	}
	augmentFromOTELPipelines(data, &cfg)

	return &cfg, nil
}

type otelPipelineConfig struct {
	Exporters map[string]otelExporter `yaml:"exporters"`
	Service   struct {
		Pipelines map[string]otelServicePipeline `yaml:"pipelines"`
	} `yaml:"service"`
}

type otelExporter struct {
	Type      string   `yaml:"type"`
	Endpoints []string `yaml:"endpoints"`
	Hosts     []string `yaml:"hosts"`
}

type otelServicePipeline struct {
	Receivers []string `yaml:"receivers"`
	Exporters []string `yaml:"exporters"`
}

// augmentFromOTELPipelines maps OTel-style pipelines into ElasticAgentConfig fields.
func augmentFromOTELPipelines(data []byte, cfg *ElasticAgentConfig) {
	if cfg == nil {
		return
	}

	var otelCfg otelPipelineConfig
	if err := yaml.Unmarshal(data, &otelCfg); err != nil {
		return
	}
	if len(otelCfg.Service.Pipelines) == 0 && len(otelCfg.Exporters) == 0 {
		return
	}
	if cfg.Outputs == nil {
		cfg.Outputs = map[string]ElasticOutput{}
	}

	for exporterName, exporter := range otelCfg.Exporters {
		key := strings.TrimSpace(exporterName)
		if key == "" {
			continue
		}
		if _, exists := cfg.Outputs[key]; exists {
			continue
		}
		hosts := exporter.Hosts
		if len(hosts) == 0 {
			hosts = exporter.Endpoints
		}
		cfg.Outputs[key] = ElasticOutput{
			Type:  nonEmpty(exporter.Type, segmentBeforeSlash(key)),
			Hosts: hosts,
		}
	}

	existing := make(map[string]struct{}, len(cfg.Inputs))
	for _, in := range cfg.Inputs {
		existing[inputKey(in.ID, in.UseOutput)] = struct{}{}
	}

	for _, servicePipeline := range otelCfg.Service.Pipelines {
		useOutput := firstNonEmpty(servicePipeline.Exporters...)
		if useOutput == "" {
			useOutput = "default"
		}
		for _, receiver := range servicePipeline.Receivers {
			id := strings.TrimSpace(receiver)
			if id == "" {
				continue
			}
			key := inputKey(id, useOutput)
			if _, seen := existing[key]; seen {
				continue
			}
			cfg.Inputs = append(cfg.Inputs, ElasticInput{
				ID:        id,
				Type:      segmentBeforeSlash(id),
				Enabled:   true,
				UseOutput: useOutput,
			})
			existing[key] = struct{}{}
		}
	}
}

func inputKey(id, output string) string {
	return strings.TrimSpace(id) + "\x00" + strings.TrimSpace(output)
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

func nonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	return nonEmpty(values...)
}
