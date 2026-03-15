package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestElasticAgent(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run Elastic Agent integration test")
	}

	repoRoot := filepath.Join("..", "..")
	var output bytes.Buffer
	deadline := time.Now().Add(90 * time.Second)
	var runErr error
	for time.Now().Before(deadline) {
		cmd := exec.Command(
			"go", "run", "./cmd/agent-cli", "status",
			"--agent", "elastic-agent",
			"--format", "json",
			"--elastic-config", "test/integration/elastic-agent.yml",
			"--elastic-url", "http://127.0.0.1:6791",
		)
		cmd.Dir = repoRoot
		cmd.Env = os.Environ()
		output.Reset()
		cmd.Stdout = &output
		cmd.Stderr = &output
		runErr = cmd.Run()
		if runErr == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if runErr != nil {
		t.Fatalf("agent-cli status failed: %v\noutput:\n%s", runErr, output.String())
	}

	raw := strings.TrimSpace(output.String())
	if !json.Valid([]byte(raw)) {
		t.Fatalf("status output is not valid JSON:\n%s", raw)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	nodes, _ := payload["nodes"].([]any)
	if len(nodes) == 0 {
		t.Fatalf("expected non-empty pipeline nodes, got payload: %s", raw)
	}
}

func TestElasticAgentStatsEndpointReachable(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run Elastic Agent integration test")
	}

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		cmd := exec.Command("sh", "-c", "curl -fsS http://127.0.0.1:6791/stats >/dev/null")
		if err := cmd.Run(); err == nil {
			return
		}
		time.Sleep(1 * time.Second)
	}

	t.Fatalf("elastic-agent stats endpoint did not become reachable on time")
}
