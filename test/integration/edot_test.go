package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestEDOTCollector(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run EDOT integration test")
	}

	repoRoot := filepath.Join("..", "..")
	binaryPath := filepath.Join(t.TempDir(), "agent-cli-test")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/agent-cli")
	buildCmd.Dir = repoRoot
	buildCmd.Env = os.Environ()
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build agent-cli binary failed: %v\noutput:\n%s", err, string(out))
	}

	var output bytes.Buffer
	deadline := time.Now().Add(90 * time.Second)
	var runErr error
	for time.Now().Before(deadline) {
		cmd := exec.Command(
			binaryPath, "status",
			"--agent", "edot",
			"--format", "json",
			"--edot-config", "test/integration/edot-collector.yml",
			"--edot-zpages-url", "http://127.0.0.1:55679",
			"--edot-metrics-url", "http://127.0.0.1:8888/metrics",
			"--edot-health-url", "http://127.0.0.1:13133/",
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
		t.Fatalf("agent-cli status failed for edot: %v\noutput:\n%s", runErr, output.String())
	}

	raw := strings.TrimSpace(output.String())
	if !json.Valid([]byte(raw)) {
		t.Fatalf("status output is not valid JSON:\n%s", raw)
	}
	if !strings.Contains(raw, `"kind":"receiver"`) && !strings.Contains(raw, `"kind": "receiver"`) {
		t.Fatalf("expected receiver nodes in EDOT payload: %s", raw)
	}
}
