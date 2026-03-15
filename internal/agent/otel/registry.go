package otel

import (
	"strings"

	"github.com/alvarolobato/agent-cli/internal/config"
)

// ComponentDescriptor provides display metadata for known OTel components.
type ComponentDescriptor struct {
	Name       string
	Kind       string
	Categories []config.OTelPipelineType
	Icon       string
	Known      bool
}

var knownComponents = map[string]ComponentDescriptor{
	"otlpreceiver": {
		Name:       "OTLP receiver",
		Kind:       "receiver",
		Categories: []config.OTelPipelineType{config.OTelPipelineTypeTrace, config.OTelPipelineTypeMetrics, config.OTelPipelineTypeLogs},
		Icon:       "receiver",
		Known:      true,
	},
	"batchprocessor": {
		Name:       "Batch processor",
		Kind:       "processor",
		Categories: []config.OTelPipelineType{config.OTelPipelineTypeTrace, config.OTelPipelineTypeMetrics, config.OTelPipelineTypeLogs},
		Icon:       "processor",
		Known:      true,
	},
	"debugexporter": {
		Name:       "Debug exporter",
		Kind:       "exporter",
		Categories: []config.OTelPipelineType{config.OTelPipelineTypeTrace, config.OTelPipelineTypeMetrics, config.OTelPipelineTypeLogs},
		Icon:       "exporter",
		Known:      true,
	},
	"prometheusreceiver": {
		Name:       "Prometheus receiver",
		Kind:       "receiver",
		Categories: []config.OTelPipelineType{config.OTelPipelineTypeMetrics},
		Icon:       "receiver",
		Known:      true,
	},
}

// LookupComponentDescriptor returns a friendly descriptor for OTel components.
// Unknown components fall back to raw names with a generic icon.
func LookupComponentDescriptor(kind, componentName, pipelineName string) ComponentDescriptor {
	componentName = strings.TrimSpace(componentName)
	kind = strings.TrimSpace(kind)

	for _, key := range registryLookupKeys(kind, componentName) {
		if descriptor, ok := knownComponents[key]; ok {
			return descriptor
		}
	}

	return ComponentDescriptor{
		Name:       componentName,
		Kind:       kind,
		Categories: []config.OTelPipelineType{pipelineTypeFromName(pipelineName)},
		Icon:       "generic",
		Known:      false,
	}
}

func registryLookupKeys(kind, componentName string) []string {
	trimmed := strings.TrimSpace(componentName)
	base := componentBaseName(trimmed)
	out := []string{
		normalizeKey(base),
		normalizeKey(base + kind),
		normalizeKey(trimmed),
		normalizeKey(trimmed + kind),
	}

	// Common aliases from service pipeline component IDs.
	switch normalizeKey(base) {
	case "otlp":
		out = append(out, "otlpreceiver")
	case "batch":
		out = append(out, "batchprocessor")
	case "debug":
		out = append(out, "debugexporter")
	case "prometheus":
		out = append(out, "prometheusreceiver")
	}

	return out
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", ""))
}

func componentBaseName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, "/"); idx > 0 {
		return trimmed[:idx]
	}
	return trimmed
}

func pipelineTypeFromName(name string) config.OTelPipelineType {
	switch componentBaseName(name) {
	case "trace", "traces":
		return config.OTelPipelineTypeTrace
	case "metrics":
		return config.OTelPipelineTypeMetrics
	case "logs":
		return config.OTelPipelineTypeLogs
	default:
		return config.OTelPipelineTypeUnknown
	}
}
