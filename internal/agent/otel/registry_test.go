package otel

import (
	"testing"

	"github.com/alvarolobato/agent-cli/internal/config"
)

func TestLookupComponentDescriptorKnownComponent(t *testing.T) {
	got := LookupComponentDescriptor("receiver", "otlp", "traces")
	if !got.Known {
		t.Fatalf("expected known OTLP receiver descriptor")
	}
	if got.Name != "OTLP receiver" {
		t.Fatalf("expected friendly name OTLP receiver, got %q", got.Name)
	}
	if got.Kind != "receiver" {
		t.Fatalf("expected receiver kind, got %q", got.Kind)
	}
	if got.Icon != "receiver" {
		t.Fatalf("expected receiver icon, got %q", got.Icon)
	}
}

func TestLookupComponentDescriptorUnknownComponent(t *testing.T) {
	got := LookupComponentDescriptor("processor", "mycustomprocessor", "logs")
	if got.Known {
		t.Fatalf("expected unknown component descriptor")
	}
	if got.Name != "mycustomprocessor" {
		t.Fatalf("expected raw component name fallback, got %q", got.Name)
	}
	if got.Icon != "generic" {
		t.Fatalf("expected generic icon for unknown component, got %q", got.Icon)
	}
	if len(got.Categories) != 1 || got.Categories[0] != config.OTelPipelineTypeLogs {
		t.Fatalf("expected logs category fallback, got %#v", got.Categories)
	}
}
