package config

import (
	"fmt"
	"os"

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

	return &cfg, nil
}
