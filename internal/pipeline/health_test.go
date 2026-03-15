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
