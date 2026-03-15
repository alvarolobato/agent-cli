package elasticagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultStatusURL = "http://localhost:6791"

// Client provides read-only access to Elastic Agent status APIs.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// StatusResponse is the payload returned by /api/status.
type StatusResponse struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Status     RuntimeStatus   `json:"status"`
	Components []ComponentInfo `json:"components"`
}

// RuntimeStatus is the common EA status shape.
type RuntimeStatus struct {
	Overall string `json:"overall"`
	Message string `json:"message"`
}

// ComponentInfo represents one Elastic Agent component entry.
type ComponentInfo struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Status RuntimeStatus `json:"status"`
	Units  []UnitInfo    `json:"units,omitempty"`
}

// UnitInfo represents a component sub-unit in Elastic Agent status.
type UnitInfo struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewClient returns an Elastic Agent status API client.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultStatusURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// GetStatus reads the full status payload from /api/status.
func (c *Client) GetStatus(ctx context.Context) (*StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/status", nil)
	if err != nil {
		return nil, fmt.Errorf("build elastic agent status request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request elastic agent status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return c.getStatusFromStats(ctx)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elastic agent status api returned %s", resp.Status)
	}

	var out StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode elastic agent status response: %w", err)
	}

	return &out, nil
}

func (c *Client) getStatusFromStats(ctx context.Context) (*StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/stats", nil)
	if err != nil {
		return nil, fmt.Errorf("build elastic agent stats request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request elastic agent stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elastic agent stats api returned %s", resp.Status)
	}

	var payload struct {
		Beat struct {
			Info struct {
				EphemeralID string `json:"ephemeral_id"`
				Name        string `json:"name"`
			} `json:"info"`
		} `json:"beat"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode elastic agent stats response: %w", err)
	}

	name := strings.TrimSpace(payload.Beat.Info.Name)
	if name == "" {
		name = "elastic-agent"
	}

	return &StatusResponse{
		ID:   strings.TrimSpace(payload.Beat.Info.EphemeralID),
		Name: name,
		Status: RuntimeStatus{
			Overall: "HEALTHY",
			Message: "derived from /stats fallback",
		},
		Components: nil,
	}, nil
}

// GetComponents returns component status entries from /api/status.
func (c *Client) GetComponents(ctx context.Context) ([]ComponentInfo, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}
	return status.Components, nil
}
