package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Snapshot is the normalized Beat metrics view used by the Elastic adapter.
type Snapshot struct {
	EventsInPerSec  float64
	EventsOutPerSec float64
	ErrorCount      float64
	DropCount       float64
}

// Collector scrapes Beat internal HTTP metrics endpoints.
type Collector struct {
	httpClient *http.Client
}

// NewCollector returns a Beat metrics collector with sane defaults.
func NewCollector(httpClient *http.Client) *Collector {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Collector{httpClient: httpClient}
}

// CollectBeatStats reads and maps a Beat /stats payload into Snapshot metrics.
func (c *Collector) CollectBeatStats(ctx context.Context, endpoint string) (*Snapshot, error) {
	url := strings.TrimSpace(endpoint)
	if url == "" {
		return nil, fmt.Errorf("beat stats endpoint is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build beat stats request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request beat stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("beat stats endpoint returned %s", resp.Status)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode beat stats response: %w", err)
	}

	eventsIn, _ := lookupNumber(payload, "libbeat", "pipeline", "events", "published")
	if eventsIn == 0 {
		eventsIn, _ = lookupNumber(payload, "libbeat", "pipeline", "events", "total")
	}
	eventsOut, _ := lookupNumber(payload, "libbeat", "output", "events", "acked")
	errors, _ := lookupNumber(payload, "libbeat", "output", "events", "failed")

	drops := sumNestedEventValue(payload, "output", "dropped")
	if drops == 0 {
		drops, _ = lookupNumber(payload, "libbeat", "output", "events", "dropped")
	}

	return &Snapshot{
		EventsInPerSec:  eventsIn,
		EventsOutPerSec: eventsOut,
		ErrorCount:      errors,
		DropCount:       drops,
	}, nil
}

func lookupNumber(payload map[string]interface{}, keys ...string) (float64, bool) {
	var cur interface{} = payload
	for _, k := range keys {
		next, ok := cur.(map[string]interface{})
		if !ok {
			return 0, false
		}
		cur, ok = next[k]
		if !ok {
			return 0, false
		}
	}

	n, ok := cur.(float64)
	if !ok {
		return 0, false
	}
	return n, true
}

func sumNestedEventValue(payload map[string]interface{}, section, field string) float64 {
	root, ok := payload[section].(map[string]interface{})
	if !ok {
		return 0
	}

	total := 0.0
	for _, raw := range root {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		events, ok := entry["events"].(map[string]interface{})
		if !ok {
			continue
		}
		if value, ok := events[field].(float64); ok {
			total += value
		}
	}
	return total
}
