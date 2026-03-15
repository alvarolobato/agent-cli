package pipeline

import "testing"

func TestAssessHealth(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want HealthStatus
	}{
		{
			name: "healthy state",
			node: &Node{Status: HealthStatus("HEALTHY")},
			want: Healthy,
		},
		{
			name: "degraded state",
			node: &Node{Status: HealthStatus("WARNING")},
			want: Degraded,
		},
		{
			name: "error state",
			node: &Node{Status: HealthStatus("FAILED")},
			want: Error,
		},
		{
			name: "disabled state",
			node: &Node{Status: HealthStatus("DISABLED")},
			want: Disabled,
		},
		{
			name: "unknown state",
			node: &Node{Status: HealthStatus("SOMETHING_ELSE")},
			want: Unknown,
		},
		{
			name: "nil node",
			node: nil,
			want: Unknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AssessHealth(tc.node)
			if got != tc.want {
				t.Fatalf("AssessHealth() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapOTelRuntimeStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   HealthStatus
	}{
		{name: "ok", status: "StatusOK", want: Healthy},
		{name: "recoverable", status: "StatusRecoverableError", want: Degraded},
		{name: "fatal", status: "StatusFatalError", want: Error},
		{name: "stopped", status: "StatusStopped", want: Disabled},
		{name: "unknown", status: "StatusUnknown", want: Unknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MapOTelRuntimeStatus(tc.status)
			if got != tc.want {
				t.Fatalf("MapOTelRuntimeStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAssessOTelComponentHealthCoversAllStates(t *testing.T) {
	tests := []struct {
		name       string
		current    HealthStatus
		sendFailed float64
		dropped    float64
		enabled    bool
		want       HealthStatus
	}{
		{name: "healthy", current: Healthy, enabled: true, want: Healthy},
		{name: "degraded by drops", current: Healthy, dropped: 1, enabled: true, want: Degraded},
		{name: "error by send failures", current: Healthy, sendFailed: 1, enabled: true, want: Error},
		{name: "disabled", current: Healthy, enabled: false, want: Disabled},
		{name: "unknown", current: Unknown, sendFailed: 0, dropped: 0, enabled: true, want: Unknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AssessOTelComponentHealth(tc.current, tc.sendFailed, tc.dropped, tc.enabled)
			if got != tc.want {
				t.Fatalf("AssessOTelComponentHealth() = %q, want %q", got, tc.want)
			}
		})
	}
}
