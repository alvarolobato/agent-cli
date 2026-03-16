package discovery

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type endpointProbe struct {
	agentType string
	url       string
	checkPath string
	key       string
}

var defaultEndpoints = []endpointProbe{
	{agentType: "elastic-agent", url: "http://localhost:6791", checkPath: "/api/status", key: "status"},
	{agentType: "otel", url: "http://localhost:55679", checkPath: "/debug/pipelinez", key: "zpages"},
	{agentType: "edot", url: "http://localhost:55679", checkPath: "/debug/pipelinez", key: "zpages"},
	{agentType: "otel", url: "http://localhost:13133", checkPath: "/", key: "health"},
	{agentType: "edot", url: "http://localhost:13133", checkPath: "/", key: "health"},
	{agentType: "otel", url: "http://localhost:8888", checkPath: "/metrics", key: "metrics"},
	{agentType: "edot", url: "http://localhost:8888", checkPath: "/metrics", key: "metrics"},
}

type probeHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type portProber struct {
	client    probeHTTPClient
	endpoints []endpointProbe
}

// NewPortProber creates a port-based discovery strategy.
func NewPortProber() Strategy {
	return &portProber{
		client:    &http.Client{Timeout: 500 * time.Millisecond},
		endpoints: defaultEndpoints,
	}
}

func NewPortProberWithClient(client probeHTTPClient, endpoints []endpointProbe) Strategy {
	if client == nil {
		client = &http.Client{Timeout: 500 * time.Millisecond}
	}
	if endpoints == nil {
		endpoints = defaultEndpoints
	}
	return &portProber{client: client, endpoints: endpoints}
}

func (s *portProber) Discover(ctx context.Context) ([]DiscoveredAgent, error) {
	byType := map[string]DiscoveredAgent{}
	for _, endpoint := range s.endpoints {
		ok := s.probeEndpoint(ctx, endpoint)
		if !ok {
			continue
		}
		agent := byType[endpoint.agentType]
		if agent.AgentType == "" {
			agent = DiscoveredAgent{
				AgentType: endpoint.agentType,
				Endpoints: map[string]string{},
				Source:    "port",
			}
		}
		agent.Endpoints[endpoint.key] = endpoint.url
		byType[endpoint.agentType] = agent
	}

	out := make([]DiscoveredAgent, 0, len(byType))
	for _, a := range byType {
		out = append(out, a)
	}
	return out, nil
}

func (s *portProber) probeEndpoint(ctx context.Context, endpoint endpointProbe) bool {
	target, err := endpointURL(endpoint.url, endpoint.checkPath)
	if err != nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func endpointURL(base string, checkPath string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if checkPath == "" {
		return parsed.String(), nil
	}
	if strings.HasPrefix(checkPath, "/") {
		parsed.Path = checkPath
		return parsed.String(), nil
	}
	parsed.Path = "/" + checkPath
	return parsed.String(), nil
}
