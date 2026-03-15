package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const defaultZPagesURL = "http://localhost:55679"

// ZPagesClient reads OTel zpages endpoints.
type ZPagesClient struct {
	baseURL    string
	httpClient *http.Client
}

// PipelineTopology represents active collector pipelines and component status.
type PipelineTopology struct {
	Pipelines       []PipelineStatus
	TracezReachable bool
}

// PipelineStatus is one OTel pipeline with its component groups.
type PipelineStatus struct {
	Name       string
	Receivers  []ComponentStatus
	Processors []ComponentStatus
	Exporters  []ComponentStatus
}

// ComponentStatus is one runtime component entry from zpages.
type ComponentStatus struct {
	ID     string
	Kind   string
	Status string
	Error  string
}

// NewZPagesClient creates a client for collector zpages endpoints.
func NewZPagesClient(baseURL string, httpClient *http.Client) *ZPagesClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultZPagesURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &ZPagesClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// GetPipelineTopology reads /debug/pipelinez and /debug/tracez to return active topology.
func (c *ZPagesClient) GetPipelineTopology(ctx context.Context) (*PipelineTopology, error) {
	body, contentType, err := c.get(ctx, "/debug/pipelinez")
	if err != nil {
		return nil, fmt.Errorf("request zpages pipelinez: %w", err)
	}

	pipelines, err := parsePipelinez(body, contentType)
	if err != nil {
		return nil, fmt.Errorf("parse zpages pipelinez: %w", err)
	}

	tracezReachable := c.isTracezReachable(ctx)

	return &PipelineTopology{
		Pipelines:       pipelines,
		TracezReachable: tracezReachable,
	}, nil
}

func (c *ZPagesClient) get(ctx context.Context, path string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request %s: %w", path, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%s returned %s", path, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read %s response body: %w", path, err)
	}

	return body, resp.Header.Get("Content-Type"), nil
}

func (c *ZPagesClient) isTracezReachable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/debug/tracez", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode == http.StatusOK
}

func parsePipelinez(body []byte, contentType string) ([]PipelineStatus, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("empty response")
	}

	if strings.Contains(strings.ToLower(contentType), "json") || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return parsePipelinezJSON(body)
	}
	return parsePipelinezHTML(trimmed)
}

func parsePipelinezJSON(body []byte) ([]PipelineStatus, error) {
	var payload struct {
		Pipelines []struct {
			Name       string            `json:"name"`
			PipelineID string            `json:"pipeline_id"`
			Receivers  []ComponentStatus `json:"receivers"`
			Processors []ComponentStatus `json:"processors"`
			Exporters  []ComponentStatus `json:"exporters"`
		} `json:"pipelines"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	out := make([]PipelineStatus, 0, len(payload.Pipelines))
	for _, p := range payload.Pipelines {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = strings.TrimSpace(p.PipelineID)
		}
		if name == "" {
			continue
		}

		out = append(out, PipelineStatus{
			Name:       name,
			Receivers:  normalizeComponents("receiver", p.Receivers),
			Processors: normalizeComponents("processor", p.Processors),
			Exporters:  normalizeComponents("exporter", p.Exporters),
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no pipelines found")
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func normalizeComponents(defaultKind string, components []ComponentStatus) []ComponentStatus {
	out := make([]ComponentStatus, 0, len(components))
	for _, c := range components {
		id := strings.TrimSpace(c.ID)
		if id == "" {
			continue
		}
		kind := strings.TrimSpace(c.Kind)
		if kind == "" {
			kind = defaultKind
		}
		out = append(out, ComponentStatus{
			ID:     id,
			Kind:   kind,
			Status: strings.TrimSpace(c.Status),
			Error:  strings.TrimSpace(c.Error),
		})
	}
	return out
}

func parsePipelinezHTML(html string) ([]PipelineStatus, error) {
	// HTML fallback only supports a deterministic mock shape using data attributes.
	rowRe := regexp.MustCompile(`(?is)<tr[^>]*data-pipeline="([^"]+)"[^>]*data-kind="([^"]+)"[^>]*data-component="([^"]+)"[^>]*data-status="([^"]*)"[^>]*data-error="([^"]*)"[^>]*>`)
	matches := rowRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return parsePipelinezCollectorHTML(html)
	}

	type pipelineParts struct {
		receivers  []ComponentStatus
		processors []ComponentStatus
		exporters  []ComponentStatus
	}
	byPipeline := map[string]*pipelineParts{}

	for _, m := range matches {
		pipelineName := strings.TrimSpace(m[1])
		kind := strings.ToLower(strings.TrimSpace(m[2]))
		componentID := strings.TrimSpace(m[3])
		status := strings.TrimSpace(m[4])
		errMsg := strings.TrimSpace(m[5])

		if pipelineName == "" || componentID == "" {
			continue
		}

		p := byPipeline[pipelineName]
		if p == nil {
			p = &pipelineParts{}
			byPipeline[pipelineName] = p
		}

		component := ComponentStatus{
			ID:     componentID,
			Kind:   kind,
			Status: status,
			Error:  errMsg,
		}

		switch kind {
		case "receiver", "receivers":
			p.receivers = append(p.receivers, component)
		case "processor", "processors":
			p.processors = append(p.processors, component)
		case "exporter", "exporters":
			p.exporters = append(p.exporters, component)
		}
	}

	if len(byPipeline) == 0 {
		return nil, fmt.Errorf("no pipeline components parsed from html")
	}

	names := make([]string, 0, len(byPipeline))
	for name := range byPipeline {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]PipelineStatus, 0, len(names))
	for _, name := range names {
		parts := byPipeline[name]
		out = append(out, PipelineStatus{
			Name:       name,
			Receivers:  parts.receivers,
			Processors: parts.processors,
			Exporters:  parts.exporters,
		})
	}

	return out, nil
}

func parsePipelinezCollectorHTML(html string) ([]PipelineStatus, error) {
	linkRe := regexp.MustCompile(`zpipelinename=([^"&]+)&zcomponentname=([^"&]+)&zcomponentkind=([^"&]+)`)
	matches := linkRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no pipeline rows found in html")
	}

	type pipelineParts struct {
		receivers  []ComponentStatus
		processors []ComponentStatus
		exporters  []ComponentStatus
		seen       map[string]bool
	}

	byPipeline := map[string]*pipelineParts{}
	for _, match := range matches {
		pipelineName, _ := url.QueryUnescape(strings.TrimSpace(match[1]))
		componentName, _ := url.QueryUnescape(strings.TrimSpace(match[2]))
		kind, _ := url.QueryUnescape(strings.TrimSpace(match[3]))
		kind = strings.ToLower(kind)
		if pipelineName == "" || componentName == "" || kind == "" {
			continue
		}

		parts := byPipeline[pipelineName]
		if parts == nil {
			parts = &pipelineParts{seen: map[string]bool{}}
			byPipeline[pipelineName] = parts
		}
		component := ComponentStatus{
			ID:   componentName,
			Kind: kind,
		}
		componentKey := kind + "|" + componentName
		if parts.seen[componentKey] {
			continue
		}
		parts.seen[componentKey] = true

		switch kind {
		case "receiver":
			parts.receivers = append(parts.receivers, component)
		case "processor":
			parts.processors = append(parts.processors, component)
		case "exporter":
			parts.exporters = append(parts.exporters, component)
		}
	}

	names := make([]string, 0, len(byPipeline))
	for name := range byPipeline {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]PipelineStatus, 0, len(names))
	for _, name := range names {
		parts := byPipeline[name]
		out = append(out, PipelineStatus{
			Name:       name,
			Receivers:  parts.receivers,
			Processors: parts.processors,
			Exporters:  parts.exporters,
		})
	}
	return out, nil
}
