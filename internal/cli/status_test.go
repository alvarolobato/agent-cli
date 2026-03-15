package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

func TestStatusPipelineElasticAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ea-1",
			"name":"standalone-agent",
			"status":{"overall":"HEALTHY","message":"ok"},
			"components":[
				{"id":"filestream-default","name":"filestream","status":{"overall":"HEALTHY","message":"ok"}},
				{"id":"output-default","name":"output-default","status":{"overall":"DEGRADED","message":"slow"}},
				{"id":"output-monitoring","name":"output-monitoring","status":{"overall":"FAILED","message":"failed"}}
			]
		}`))
	}))
	defer server.Close()

	cmd := newStatusCommand()
	pipe, err := statusPipeline(cmd, "elastic-agent", "../../test/fixtures/elastic-agent.yml", server.URL)
	if err != nil {
		t.Fatalf("statusPipeline() error = %v", err)
	}

	if len(pipe.Nodes) == 0 {
		t.Fatalf("expected nodes in pipeline")
	}
	for _, n := range pipe.Nodes {
		if n.Kind == "output" && n.Label == "default" && n.Status != pipeline.Degraded {
			t.Fatalf("expected output default degraded, got %q", n.Status)
		}
	}
}

func TestStatusCommandJSONElasticAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ea-1",
			"name":"standalone-agent",
			"status":{"overall":"HEALTHY","message":"ok"},
			"components":[]
		}`))
	}))
	defer server.Close()

	cmd := newStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--agent", "elastic-agent",
		"--format", "json",
		"--elastic-config", "../../test/fixtures/elastic-agent.yml",
		"--elastic-url", server.URL,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload := strings.TrimSpace(out.String())
	if !json.Valid([]byte(payload)) {
		t.Fatalf("expected valid JSON output, got %s", payload)
	}
}
