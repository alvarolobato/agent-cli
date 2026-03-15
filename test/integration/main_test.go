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
		"127.0.0.1:13133", // otelcol health
		"127.0.0.1:55679", // edot zpages
	}

	for _, target := range targets {
		conn, err := net.DialTimeout("tcp", target, 2*time.Second)
		if err != nil {
			t.Fatalf("service %s is not reachable: %v", target, err)
		}
		if err := conn.Close(); err != nil {
			t.Fatalf("service %s close failed: %v", target, err)
		}
	}

	t.Logf("all integration targets reachable: %s", fmt.Sprint(targets))
}
