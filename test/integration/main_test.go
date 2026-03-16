package integration

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestServicesReachable(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run docker-compose integration reachability checks")
	}

	targets := []string{
		"127.0.0.1:6791",  // elastic-agent
		"127.0.0.1:13133", // edot health_check
		"127.0.0.1:55679", // edot zpages
		"127.0.0.1:13134", // otel health_check
		"127.0.0.1:55680", // otel zpages
		"127.0.0.1:8889",  // otel metrics
	}

	for _, target := range targets {
		deadline := time.Now().Add(15 * time.Second)
		var lastErr error

		for time.Now().Before(deadline) {
			conn, err := net.DialTimeout("tcp", target, 2*time.Second)
			if err == nil {
				if closeErr := conn.Close(); closeErr != nil {
					t.Fatalf("service %s close failed: %v", target, closeErr)
				}
				lastErr = nil
				break
			}

			lastErr = err
			time.Sleep(250 * time.Millisecond)
		}

		if lastErr != nil {
			t.Fatalf("service %s is not reachable after retries: %v", target, lastErr)
		}
	}

	t.Logf("all integration targets reachable: %s", fmt.Sprint(targets))
}
