package config

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParseElasticAgentConfigWithInspectUsesInspectOutput(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`
outputs:
  default:
    type: elasticsearch
inputs:
  - id: from-inspect
    type: filestream
    use_output: default
`), nil
	}

	result, err := ParseElasticAgentConfigWithInspect(context.Background(), "../../test/fixtures/elastic-agent.yml", runner)
	if err != nil {
		t.Fatalf("ParseElasticAgentConfigWithInspect() error = %v", err)
	}
	if result.Source != configSourceInspect {
		t.Fatalf("source = %q, want %q", result.Source, configSourceInspect)
	}
	if len(result.Config.Inputs) != 1 || result.Config.Inputs[0].ID != "from-inspect" {
		t.Fatalf("unexpected inspect config inputs: %#v", result.Config.Inputs)
	}
}

func TestParseElasticAgentConfigWithInspectFallsBackOnRunnerError(t *testing.T) {
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("inspect unavailable")
	}

	result, err := ParseElasticAgentConfigWithInspect(context.Background(), "../../test/fixtures/elastic-agent.yml", runner)
	if err != nil {
		t.Fatalf("ParseElasticAgentConfigWithInspect() error = %v", err)
	}
	if result.Source != configSourceFiles {
		t.Fatalf("source = %q, want %q", result.Source, configSourceFiles)
	}
	if len(result.Config.Inputs) == 0 {
		t.Fatalf("expected file-based config inputs")
	}
}

func TestParseElasticAgentConfigWithInspectPassesInspectArgs(t *testing.T) {
	var gotName string
	var gotArgs []string
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return []byte("outputs: {}\ninputs: []\n"), nil
	}

	_, err := ParseElasticAgentConfigWithInspect(context.Background(), "/opt/Elastic/Agent/elastic-agent.yml", runner)
	if err != nil {
		t.Fatalf("ParseElasticAgentConfigWithInspect() error = %v", err)
	}

	if gotName == "" {
		t.Fatalf("expected inspect binary to be invoked")
	}
	joined := strings.Join(gotArgs, " ")
	for _, token := range []string{"inspect", "--path.home", "/opt/Elastic/Agent", "--path.config", "-c", "elastic-agent.yml"} {
		if !strings.Contains(joined, token) {
			t.Fatalf("inspect args missing %q: %q", token, joined)
		}
	}
}
