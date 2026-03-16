package discovery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestProcessScannerRecognizesEA93Tree(t *testing.T) {
	provider := func(context.Context) ([]ProcessInfo, error) {
		return []ProcessInfo{
			{PID: 100, Name: "elastic-agent"},
			{
				PID:  200,
				PPID: 100,
				Name: "elastic-otel-collector",
				Args: []string{"--supervised", "--config", "/etc/edot/config.yaml"},
			},
		}, nil
	}

	strategy := NewProcessScannerWithProvider(provider)
	agents, err := strategy.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].AgentType != "elastic-agent" {
		t.Fatalf("expected elastic-agent, got %q", agents[0].AgentType)
	}
	if len(agents[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(agents[0].Children))
	}
	if agents[0].Children[0].Role != "otel-collector" {
		t.Fatalf("expected otel-collector child role, got %q", agents[0].Children[0].Role)
	}
}

func TestProcessScannerRecognizesEA92Tree(t *testing.T) {
	provider := func(context.Context) ([]ProcessInfo, error) {
		return []ProcessInfo{
			{PID: 714, Name: "elastic-agent"},
			{PID: 1091, PPID: 714, Name: "elastic-agent", Args: []string{"otel", "--supervised"}},
			{PID: 1443, PPID: 714, Name: "agentbeat", Args: []string{"metricbeat", "-E", "path.data=/opt/run/metrics-default"}},
			{PID: 1722, PPID: 714, Name: "agentbeat", Args: []string{"filebeat", "-E", "path.data=/opt/run/log-default"}},
		}, nil
	}
	strategy := NewProcessScannerWithProvider(provider)
	agents, err := strategy.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if got := len(agents[0].Children); got != 3 {
		t.Fatalf("expected 3 children, got %d", got)
	}
}

func TestProcessScannerStandaloneEDOT(t *testing.T) {
	provider := func(context.Context) ([]ProcessInfo, error) {
		return []ProcessInfo{
			{PID: 500, Name: "elastic-otel-collector", Args: []string{"--config", "/etc/edot/config.yaml"}},
		}, nil
	}
	strategy := NewProcessScannerWithProvider(provider)
	agents, err := strategy.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 1 || agents[0].AgentType != "edot" {
		t.Fatalf("expected one edot agent, got %+v", agents)
	}
	if agents[0].ConfigPath != "/etc/edot/config.yaml" {
		t.Fatalf("expected config path extracted, got %q", agents[0].ConfigPath)
	}
	if len(agents[0].Children) != 0 {
		t.Fatalf("expected standalone EDOT without children, got %+v", agents[0].Children)
	}
}

func TestClassifyProcessKnownBinaries(t *testing.T) {
	cases := []struct {
		name      string
		proc      ProcessInfo
		wantType  string
		wantRole  string
		wantChild bool
	}{
		{
			name:      "elastic-otel-collector maps to edot",
			proc:      ProcessInfo{Name: "elastic-otel-collector"},
			wantType:  "edot",
			wantRole:  "otel-collector",
			wantChild: true,
		},
		{
			name:      "elastic-agent otel child",
			proc:      ProcessInfo{Name: "elastic-agent", Args: []string{"otel", "--supervised"}},
			wantType:  "edot",
			wantRole:  "otel-collector",
			wantChild: true,
		},
		{
			name:      "agentbeat metricbeat child",
			proc:      ProcessInfo{Name: "agentbeat", Args: []string{"metricbeat"}},
			wantType:  "elastic-agent",
			wantRole:  "metricbeat",
			wantChild: true,
		},
		{
			name:      "agentbeat filebeat child",
			proc:      ProcessInfo{Name: "agentbeat", Args: []string{"filebeat"}},
			wantType:  "elastic-agent",
			wantRole:  "filebeat",
			wantChild: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotRole, gotChild := classifyProcess(tc.proc)
			if gotType != tc.wantType || gotRole != tc.wantRole || gotChild != tc.wantChild {
				t.Fatalf("classifyProcess() = (%q,%q,%v), want (%q,%q,%v)", gotType, gotRole, gotChild, tc.wantType, tc.wantRole, tc.wantChild)
			}
		})
	}
}

func TestParseProcessArgumentsExtractsConfigAndEndpoints(t *testing.T) {
	config, endpoints := parseProcessArguments([]string{
		"--supervised",
		"--config", "/etc/edot/config.yaml",
		"--supervised.monitoring.url=unix:///opt/agent.sock",
		"-E", "http.host=unix:///opt/beats.sock",
		"-E", "path.data=/opt/run/metrics-default",
		"--http-addr=127.0.0.1:4318",
		"--grpc-addr", "127.0.0.1:4317",
	})
	if config != "/etc/edot/config.yaml" {
		t.Fatalf("expected config path, got %q", config)
	}
	expected := map[string]string{
		"monitoring_url":  "unix:///opt/agent.sock",
		"beats_http_host": "unix:///opt/beats.sock",
		"path_data":       "/opt/run/metrics-default",
		"http_addr":       "127.0.0.1:4318",
		"grpc_addr":       "127.0.0.1:4317",
	}
	if !reflect.DeepEqual(endpoints, expected) {
		t.Fatalf("unexpected endpoints: got %+v want %+v", endpoints, expected)
	}
}

func TestParsePSLineUsesBasename(t *testing.T) {
	proc, ok := parsePSLine("1091 714 /opt/Elastic/Agent/data/elastic-agent-9.2.4/elastic-agent otel --supervised")
	if !ok {
		t.Fatalf("parsePSLine() returned !ok")
	}
	if proc.PPID != 714 {
		t.Fatalf("expected ppid 714, got %d", proc.PPID)
	}
	if proc.Name != "elastic-agent" {
		t.Fatalf("expected basename elastic-agent, got %q", proc.Name)
	}
	if len(proc.Args) == 0 || proc.Args[0] != "otel" {
		t.Fatalf("expected otel subcommand args, got %+v", proc.Args)
	}
}

func TestPortProberIncludesPrometheusForOTelAndEDOT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metrics", "/debug/pipelinez", "/", "/api/status":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	endpoints := []endpointProbe{
		{agentType: "otel", url: server.URL, checkPath: "/metrics", key: "metrics"},
		{agentType: "edot", url: server.URL, checkPath: "/metrics", key: "metrics"},
	}
	agents, err := NewPortProberWithClient(server.Client(), endpoints).Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
}

func TestPortProberReturnsDeterministicTypeOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	endpoints := []endpointProbe{
		{agentType: "otel", url: server.URL, checkPath: "/metrics", key: "metrics"},
		{agentType: "edot", url: server.URL, checkPath: "/metrics", key: "metrics"},
	}
	agents, err := NewPortProberWithClient(server.Client(), endpoints).Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].AgentType != "edot" || agents[1].AgentType != "otel" {
		t.Fatalf("expected deterministic ordering [edot, otel], got [%s, %s]", agents[0].AgentType, agents[1].AgentType)
	}
}

func TestPathScannerChecksEDOTPathsOnDarwinAndLinux(t *testing.T) {
	checked := make([]string, 0)
	stat := func(path string) error {
		checked = append(checked, path)
		return errors.New("not found")
	}
	_, _ = NewPathScannerWithRules("darwin", stat, nil).Discover(context.Background())
	_, _ = NewPathScannerWithRules("linux", stat, nil).Discover(context.Background())

	wantDarwin := "/etc/edot/config.yaml"
	wantLinuxA := "/etc/edot/config.yaml"
	wantLinuxB := "/etc/elastic-otel-collector/config.yaml"
	if !contains(checked, wantDarwin) || !contains(checked, wantLinuxA) || !contains(checked, wantLinuxB) {
		t.Fatalf("missing expected EDOT path checks, got %v", checked)
	}
}

func TestOrchestratorMergesAcrossStrategies(t *testing.T) {
	o := &Orchestrator{
		strategies: []Strategy{
			staticStrategy{agents: []DiscoveredAgent{
				{AgentType: "edot", PID: 500, Source: "process"},
			}},
			staticStrategy{agents: []DiscoveredAgent{
				{AgentType: "edot", ConfigPath: "/etc/edot/config.yaml", Source: "path"},
			}},
		},
	}
	agents, err := o.DiscoverDetailed(context.Background())
	if err != nil {
		t.Fatalf("DiscoverDetailed() error = %v", err)
	}
	if len(agents) == 0 {
		t.Fatalf("expected merged agents")
	}
}

func TestOrchestratorPrefersPathConfigOverProcessGuess(t *testing.T) {
	o := &Orchestrator{
		strategies: []Strategy{
			staticStrategy{agents: []DiscoveredAgent{
				{AgentType: "elastic-agent", ConfigPath: "/tmp/guess/elastic-agent.yml", Source: "process"},
			}},
			staticStrategy{agents: []DiscoveredAgent{
				{AgentType: "elastic-agent", ConfigPath: "/opt/Elastic/Agent/elastic-agent.yml", Source: "path"},
			}},
		},
	}
	agents, err := o.DiscoverDetailed(context.Background())
	if err != nil {
		t.Fatalf("DiscoverDetailed() error = %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected exactly one merged agent, got %d", len(agents))
	}
	if agents[0].ConfigPath != "/opt/Elastic/Agent/elastic-agent.yml" {
		t.Fatalf("expected path strategy config to win, got %q", agents[0].ConfigPath)
	}
}

func TestOrchestratorDoesNotCollapseMultipleProcessInstances(t *testing.T) {
	o := &Orchestrator{
		strategies: []Strategy{
			staticStrategy{agents: []DiscoveredAgent{
				{AgentType: "elastic-agent", PID: 100, Source: "process"},
				{AgentType: "elastic-agent", PID: 200, Source: "process"},
			}},
		},
	}

	agents, err := o.DiscoverDetailed(context.Background())
	if err != nil {
		t.Fatalf("DiscoverDetailed() error = %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected two elastic-agent instances, got %d", len(agents))
	}
}

func TestDiscoveredAgentIDIncludesPIDWhenAvailable(t *testing.T) {
	a := DiscoveredAgent{AgentType: "elastic-agent", PID: 714}
	if got := a.ID(); got != "elastic-agent:714" {
		t.Fatalf("expected PID-qualified ID, got %q", got)
	}
}

func TestProcessScannerAttachesChildrenByPPIDWhenMultipleParents(t *testing.T) {
	provider := func(context.Context) ([]ProcessInfo, error) {
		return []ProcessInfo{
			{PID: 100, Name: "elastic-agent"},
			{PID: 200, Name: "elastic-agent"},
			{PID: 201, PPID: 200, Name: "agentbeat", Args: []string{"metricbeat"}},
		}, nil
	}

	strategy := NewProcessScannerWithProvider(provider)
	agents, err := strategy.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 parent agents, got %d", len(agents))
	}

	for _, a := range agents {
		if a.PID == 200 && len(a.Children) != 1 {
			t.Fatalf("expected child attached to PID 200 parent, got %+v", a.Children)
		}
		if a.PID == 100 && len(a.Children) != 0 {
			t.Fatalf("expected no children on PID 100 parent, got %+v", a.Children)
		}
	}
}

func TestProcessScannerLeavesUnmatchedChildStandaloneWithSingleParent(t *testing.T) {
	provider := func(context.Context) ([]ProcessInfo, error) {
		return []ProcessInfo{
			{PID: 100, Name: "elastic-agent"},
			{PID: 201, PPID: 100, Name: "agentbeat", Args: []string{"metricbeat"}},
			{PID: 300, PPID: 999, Name: "elastic-otel-collector", Args: []string{"--config", "/etc/edot/config.yaml"}},
		}, nil
	}

	strategy := NewProcessScannerWithProvider(provider)
	agents, err := strategy.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected parent plus standalone child, got %d agents", len(agents))
	}

	var parent DiscoveredAgent
	var standalone DiscoveredAgent
	for _, a := range agents {
		if a.AgentType == "elastic-agent" && a.PID == 100 {
			parent = a
		}
		if a.AgentType == "edot" && a.PID == 300 {
			standalone = a
		}
	}
	if parent.PID == 0 {
		t.Fatalf("expected elastic-agent parent in results, got %+v", agents)
	}
	if len(parent.Children) != 1 {
		t.Fatalf("expected only PPID-matched child attached, got %+v", parent.Children)
	}
	if standalone.PID == 0 {
		t.Fatalf("expected unmatched child to remain standalone, got %+v", agents)
	}
}

func TestProcessScannerWithNilProviderUsesDefault(t *testing.T) {
	strategy, ok := NewProcessScannerWithProvider(nil).(*processScanner)
	if !ok {
		t.Fatalf("expected processScanner strategy type")
	}
	if strategy.listProcesses == nil {
		t.Fatalf("expected default process provider to be set")
	}
}

type staticStrategy struct {
	agents []DiscoveredAgent
}

func (s staticStrategy) Discover(context.Context) ([]DiscoveredAgent, error) {
	return s.agents, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
