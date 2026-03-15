package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
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
	mu         sync.Mutex
	previous   map[string]timedSnapshot
	now        func() time.Time
}

type timedSnapshot struct {
	takenAt  time.Time
	counters snapshotCounters
}

type snapshotCounters struct {
	eventsIn  float64
	eventsOut float64
	errors    float64
	drops     float64
}

// NewCollector returns a Beat metrics collector with sane defaults.
func NewCollector(httpClient *http.Client) *Collector {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Collector{
		httpClient: httpClient,
		previous:   map[string]timedSnapshot{},
		now:        time.Now,
	}
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

	counters := readCounters(payload)

	now := c.now()
	rates, hasPrevious := c.updateAndComputeRates(url, counters, now)
	if !hasPrevious {
		rates = snapshotCounters{}
	}

	return &Snapshot{
		EventsInPerSec:  rates.eventsIn,
		EventsOutPerSec: rates.eventsOut,
		ErrorCount:      counters.errors,
		DropCount:       counters.drops,
	}, nil
}

func (c *Collector) updateAndComputeRates(endpoint string, current snapshotCounters, now time.Time) (snapshotCounters, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	previous, ok := c.previous[endpoint]
	c.previous[endpoint] = timedSnapshot{
		takenAt:  now,
		counters: current,
	}
	if !ok {
		return snapshotCounters{}, false
	}

	seconds := now.Sub(previous.takenAt).Seconds()
	if seconds <= 0 {
		return snapshotCounters{}, true
	}

	return snapshotCounters{
		eventsIn:  positiveDeltaPerSecond(current.eventsIn, previous.counters.eventsIn, seconds),
		eventsOut: positiveDeltaPerSecond(current.eventsOut, previous.counters.eventsOut, seconds),
		errors:    positiveDeltaPerSecond(current.errors, previous.counters.errors, seconds),
		drops:     positiveDeltaPerSecond(current.drops, previous.counters.drops, seconds),
	}, true
}

func positiveDeltaPerSecond(current, previous, seconds float64) float64 {
	delta := current - previous
	if delta <= 0 || seconds <= 0 {
		return 0
	}
	return delta / seconds
}

func readCounters(payload map[string]interface{}) snapshotCounters {
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
	return snapshotCounters{
		eventsIn:  eventsIn,
		eventsOut: eventsOut,
		errors:    errors,
		drops:     drops,
	}
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
