package discovery

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ProcessInfo describes a running process that might map to an agent.
type ProcessInfo struct {
	PID     int
	PPID    int
	Name    string
	Command string
	Args    []string
}

// ProcessProvider allows process listing to be mocked in tests.
type ProcessProvider func(ctx context.Context) ([]ProcessInfo, error)

type processScanner struct {
	listProcesses ProcessProvider
}

// NewProcessScanner creates a process-based strategy.
func NewProcessScanner() Strategy {
	return &processScanner{listProcesses: defaultProcessProvider}
}

// NewProcessScannerWithProvider creates a process scanner with a custom process source.
func NewProcessScannerWithProvider(provider ProcessProvider) Strategy {
	if provider == nil {
		provider = defaultProcessProvider
	}
	return &processScanner{listProcesses: provider}
}

func (s *processScanner) Discover(ctx context.Context) ([]DiscoveredAgent, error) {
	processes, err := s.listProcesses(ctx)
	if err != nil {
		return nil, err
	}

	parents := make([]DiscoveredAgent, 0)
	children := make([]childCandidate, 0)
	standalone := make([]DiscoveredAgent, 0)

	for _, proc := range processes {
		typ, role, child := classifyProcess(proc)
		if typ == "" {
			continue
		}
		configPath, endpoints := parseProcessArguments(proc.Args)
		configPath = strings.TrimSpace(configPath)

		if child {
			children = append(children, childCandidate{
				child: DiscoveredChild{
					PID:  proc.PID,
					Name: proc.Name,
					Role: role,
					Args: append([]string{}, proc.Args...),
				},
				parentPID:  proc.PPID,
				agentType:  typ,
				configPath: configPath,
				endpoints:  endpoints,
			})
			continue
		}

		agent := DiscoveredAgent{
			AgentType:  typ,
			PID:        proc.PID,
			ConfigPath: configPath,
			Endpoints:  endpoints,
			Source:     "process",
		}
		if typ == "elastic-agent" {
			parents = append(parents, agent)
		} else {
			standalone = append(standalone, agent)
		}
	}

	if len(parents) > 0 {
		parentsByPID := make(map[int]int, len(parents))
		for i := range parents {
			parentsByPID[parents[i].PID] = i
		}

		attached := make([]bool, len(children))
		for i := range children {
			if parentIdx, ok := parentsByPID[children[i].parentPID]; ok {
				parents[parentIdx].Children = append(parents[parentIdx].Children, children[i].child)
				attached[i] = true
			}
		}

		unmatched := make([]childCandidate, 0, len(children))
		for i := range children {
			if attached[i] {
				continue
			}
			unmatched = append(unmatched, children[i])
		}
		for i := range parents {
			sortDiscoveredChildren(parents[i].Children)
		}
		return append(append(parents, standalone...), childrenToStandalone(unmatched)...), nil
	}
	return append(standalone, childrenToStandalone(children)...), nil
}

func defaultProcessProvider(ctx context.Context) ([]ProcessInfo, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return []ProcessInfo{}, nil
	}

	cmd := exec.CommandContext(ctx, "ps", "-eo", "pid,ppid,args")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	processes := make([]ProcessInfo, 0, len(lines))
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		proc, ok := parsePSLine(line)
		if ok {
			processes = append(processes, proc)
		}
	}
	return processes, nil
}

func parsePSLine(line string) (ProcessInfo, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return ProcessInfo{}, false
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return ProcessInfo{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return ProcessInfo{}, false
	}
	command := fields[2]
	args := append([]string{}, fields[3:]...)
	return ProcessInfo{
		PID:     pid,
		PPID:    ppid,
		Name:    filepath.Base(command),
		Command: command,
		Args:    args,
	}, true
}

func classifyProcess(proc ProcessInfo) (agentType string, role string, child bool) {
	name := filepath.Base(proc.Name)
	primaryArg := strings.ToLower(strings.TrimSpace(firstArg(proc.Args)))
	switch name {
	case "elastic-agent":
		if len(proc.Args) > 0 && proc.Args[0] == "otel" {
			return "edot", "otel-collector", true
		}
		return "elastic-agent", "", false
	case "elastic-otel-collector", "edot-collector":
		// EA >= 9.3 uses elastic-otel-collector as a beat runner too.
		if isBeatSubcommand(primaryArg) {
			return "elastic-agent", primaryArg, true
		}
		if hasArg(proc.Args, "--supervised") || primaryArg == "" {
			return "edot", "otel-collector", true
		}
		return "edot", "otel-collector", false
	case "agentbeat":
		if len(proc.Args) > 0 {
			switch proc.Args[0] {
			case "metricbeat":
				return "elastic-agent", "metricbeat", true
			case "filebeat":
				return "elastic-agent", "filebeat", true
			}
		}
		return "elastic-agent", "agentbeat", true
	case "elastic-endpoint":
		return "elastic-endpoint", "endpoint", false
	case "otelcol", "otelcol-contrib", "otelcorecol":
		return "otel", "otel-collector", false
	default:
		return "", "", false
	}
}

func firstArg(args []string) string {
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" || strings.HasPrefix(trimmed, "-") {
			continue
		}
		return trimmed
	}
	return ""
}

func hasArg(args []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == target {
			return true
		}
	}
	return false
}

func isBeatSubcommand(value string) bool {
	switch value {
	case "filebeat", "metricbeat", "osquerybeat", "heartbeat", "auditbeat", "packetbeat":
		return true
	default:
		return false
	}
}

func parseProcessArguments(args []string) (configPath string, endpoints map[string]string) {
	endpoints = map[string]string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--config" || arg == "-c" {
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
			continue
		}
		if strings.HasPrefix(arg, "-c=") {
			configPath = strings.TrimPrefix(arg, "-c=")
			continue
		}
		if strings.HasPrefix(arg, "--supervised.monitoring.url=") {
			endpoints["monitoring_url"] = strings.TrimPrefix(arg, "--supervised.monitoring.url=")
			continue
		}
		if arg == "--http-addr" && i+1 < len(args) {
			endpoints["http_addr"] = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--http-addr=") {
			endpoints["http_addr"] = strings.TrimPrefix(arg, "--http-addr=")
			continue
		}
		if arg == "--grpc-addr" && i+1 < len(args) {
			endpoints["grpc_addr"] = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--grpc-addr=") {
			endpoints["grpc_addr"] = strings.TrimPrefix(arg, "--grpc-addr=")
			continue
		}
		if arg == "-E" && i+1 < len(args) {
			parseEFlag(endpoints, args[i+1])
			i++
			continue
		}
		if strings.HasPrefix(arg, "-E") {
			parseEFlag(endpoints, strings.TrimPrefix(arg, "-E"))
			continue
		}
	}
	return configPath, endpoints
}

func parseEFlag(endpoints map[string]string, value string) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http.host=") {
		endpoints["beats_http_host"] = strings.TrimPrefix(value, "http.host=")
	}
	if strings.HasPrefix(value, "path.data=") {
		endpoints["path_data"] = strings.TrimPrefix(value, "path.data=")
	}
}

type childCandidate struct {
	child      DiscoveredChild
	parentPID  int
	agentType  string
	configPath string
	endpoints  map[string]string
}

func sortDiscoveredChildren(children []DiscoveredChild) {
	if len(children) < 2 {
		return
	}
	for i := 0; i < len(children)-1; i++ {
		for j := i + 1; j < len(children); j++ {
			if children[j].PID < children[i].PID {
				children[i], children[j] = children[j], children[i]
			}
		}
	}
}

func childrenToStandalone(children []childCandidate) []DiscoveredAgent {
	out := make([]DiscoveredAgent, 0, len(children))
	for _, child := range children {
		typ := child.agentType
		switch child.child.Role {
		case "metricbeat", "filebeat", "agentbeat":
			typ = "elastic-agent"
		}
		out = append(out, DiscoveredAgent{
			AgentType:  typ,
			PID:        child.child.PID,
			ConfigPath: child.configPath,
			Endpoints:  child.endpoints,
			Source:     "process",
		})
	}
	return out
}
