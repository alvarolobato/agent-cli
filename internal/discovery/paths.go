package discovery

import (
	"context"
	"os"
	"runtime"
)

type pathRule struct {
	agentType string
	paths     map[string][]string
}

var defaultPathRules = []pathRule{
	{
		agentType: "elastic-agent",
		paths: map[string][]string{
			"darwin":  {"/Library/Elastic/Agent/elastic-agent.yml"},
			"linux":   {"/opt/Elastic/Agent/elastic-agent.yml"},
			"windows": {`C:\Program Files\Elastic\Agent\elastic-agent.yml`},
		},
	},
	{
		agentType: "otel",
		paths: map[string][]string{
			"darwin": {"/etc/otelcol/config.yaml"},
			"linux":  {"/etc/otelcol/config.yaml", "/etc/otel/config.yaml"},
		},
	},
	{
		agentType: "edot",
		paths: map[string][]string{
			"darwin": {"/etc/edot/config.yaml"},
			"linux":  {"/etc/edot/config.yaml", "/etc/elastic-otel-collector/config.yaml"},
		},
	},
}

type pathScanner struct {
	osName string
	stat   func(string) error
	rules  []pathRule
}

// NewPathScanner creates a path-based discovery strategy.
func NewPathScanner() Strategy {
	return &pathScanner{
		osName: runtime.GOOS,
		stat: func(path string) error {
			_, err := os.Stat(path)
			return err
		},
		rules: defaultPathRules,
	}
}

func NewPathScannerWithRules(osName string, stat func(string) error, rules []pathRule) Strategy {
	if osName == "" {
		osName = runtime.GOOS
	}
	if stat == nil {
		stat = func(path string) error {
			_, err := os.Stat(path)
			return err
		}
	}
	if rules == nil {
		rules = defaultPathRules
	}
	return &pathScanner{
		osName: osName,
		stat:   stat,
		rules:  rules,
	}
}

func (s *pathScanner) Discover(context.Context) ([]DiscoveredAgent, error) {
	out := make([]DiscoveredAgent, 0)
	for _, rule := range s.rules {
		candidates := rule.paths[s.osName]
		for _, candidate := range candidates {
			if err := s.stat(candidate); err != nil {
				continue
			}
			out = append(out, DiscoveredAgent{
				AgentType:  rule.agentType,
				ConfigPath: candidate,
				Endpoints:  map[string]string{},
				Source:     "path",
			})
			break
		}
	}
	return out, nil
}
