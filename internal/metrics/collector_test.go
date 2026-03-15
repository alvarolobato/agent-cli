package metrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCollectBeatStatsMapsMetrics(t *testing.T) {
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stats" {
			t.Fatalf("expected /stats path, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		call++
		published := 1200
		acked := 1188
		failed := 5
		droppedOne := 2
		droppedTwo := 1
		if call > 1 {
			published = 1260
			acked = 1230
			failed = 7
			droppedOne = 3
			droppedTwo = 2
		}
		_, _ = fmt.Fprintf(w, `{
			"libbeat": {
				"pipeline": {"events": {"published": %d}},
				"output": {"events": {"acked": %d, "failed": %d}}
			},
			"output": {
				"elasticsearch": {"events": {"dropped": %d}},
				"logstash": {"events": {"dropped": %d}}
			}
		}`, published, acked, failed, droppedOne, droppedTwo)
	}))
	defer server.Close()

	collector := NewCollector(server.Client())
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	collector.now = func() time.Time { return now }

	first, err := collector.CollectBeatStats(context.Background(), server.URL+"/stats")
	if err != nil {
		t.Fatalf("CollectBeatStats(first) error = %v", err)
	}
	if first.EventsInPerSec != 0 {
		t.Fatalf("expected first sample events in rate 0, got %v", first.EventsInPerSec)
	}
	if first.EventsOutPerSec != 0 {
		t.Fatalf("expected first sample events out rate 0, got %v", first.EventsOutPerSec)
	}

	now = now.Add(2 * time.Second)
	got, err := collector.CollectBeatStats(context.Background(), server.URL+"/stats")
	if err != nil {
		t.Fatalf("CollectBeatStats() error = %v", err)
	}

	if got.EventsInPerSec != 30 {
		t.Fatalf("expected events in/sec 30, got %v", got.EventsInPerSec)
	}
	if got.EventsOutPerSec != 21 {
		t.Fatalf("expected events out/sec 21, got %v", got.EventsOutPerSec)
	}
	if got.ErrorCount != 7 {
		t.Fatalf("expected errors 7, got %v", got.ErrorCount)
	}
	if got.DropCount != 5 {
		t.Fatalf("expected drops 5, got %v", got.DropCount)
	}
}

func TestCollectBeatStatsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	collector := NewCollector(server.Client())
	if _, err := collector.CollectBeatStats(context.Background(), server.URL+"/stats"); err == nil {
		t.Fatalf("expected error when endpoint returns non-200")
	}
}
