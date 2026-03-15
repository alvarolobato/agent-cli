package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectBeatStatsMapsMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stats" {
			t.Fatalf("expected /stats path, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"libbeat": {
				"pipeline": {"events": {"published": 1200}},
				"output": {"events": {"acked": 1188, "failed": 5}}
			},
			"output": {
				"elasticsearch": {"events": {"dropped": 2}},
				"logstash": {"events": {"dropped": 1}}
			}
		}`))
	}))
	defer server.Close()

	collector := NewCollector(server.Client())
	got, err := collector.CollectBeatStats(context.Background(), server.URL+"/stats")
	if err != nil {
		t.Fatalf("CollectBeatStats() error = %v", err)
	}

	if got.EventsInPerSec != 1200 {
		t.Fatalf("expected events in 1200, got %v", got.EventsInPerSec)
	}
	if got.EventsOutPerSec != 1188 {
		t.Fatalf("expected events out 1188, got %v", got.EventsOutPerSec)
	}
	if got.ErrorCount != 5 {
		t.Fatalf("expected errors 5, got %v", got.ErrorCount)
	}
	if got.DropCount != 3 {
		t.Fatalf("expected drops 3, got %v", got.DropCount)
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
