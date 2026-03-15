package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectOTelPrometheusParsesKeyMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			t.Fatalf("expected /metrics path, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(`
# Receiver metrics
otelcol_receiver_accepted_spans{receiver="otlp",transport="grpc"} 145230
otelcol_receiver_accepted_metric_points{receiver="otlp"} 10

# Processor metrics
otelcol_processor_dropped_spans{processor="batch"} 8

# Exporter metrics
otelcol_exporter_sent_spans{exporter="elasticsearch"} 145200
otelcol_exporter_send_failed_spans{exporter="elasticsearch"} 3
`))
	}))
	defer server.Close()

	collector := NewCollector(server.Client())
	snapshot, err := collector.CollectOTelPrometheus(context.Background(), server.URL+"/metrics")
	if err != nil {
		t.Fatalf("CollectOTelPrometheus() error = %v", err)
	}

	if got := snapshot.Receivers["otlp"].Accepted; got != 145240 {
		t.Fatalf("expected receiver accepted total 145240, got %v", got)
	}
	if got := snapshot.Processors["batch"].Dropped; got != 8 {
		t.Fatalf("expected processor dropped total 8, got %v", got)
	}
	if got := snapshot.Exporters["elasticsearch"].Sent; got != 145200 {
		t.Fatalf("expected exporter sent total 145200, got %v", got)
	}
	if got := snapshot.Exporters["elasticsearch"].SendFailed; got != 3 {
		t.Fatalf("expected exporter send_failed total 3, got %v", got)
	}
}

func TestCollectOTelPrometheusHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	collector := NewCollector(server.Client())
	if _, err := collector.CollectOTelPrometheus(context.Background(), server.URL+"/metrics"); err == nil {
		t.Fatalf("expected error when endpoint returns non-200")
	}
}

func TestCollectOTelPrometheusIgnoresOptionalTimestamp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(`
otelcol_exporter_sent_spans{exporter="elasticsearch"} 145200 1711111111000
otelcol_exporter_send_failed_spans{exporter="elasticsearch"} 7 1711111111000
`))
	}))
	defer server.Close()

	collector := NewCollector(server.Client())
	snapshot, err := collector.CollectOTelPrometheus(context.Background(), server.URL+"/metrics")
	if err != nil {
		t.Fatalf("CollectOTelPrometheus() error = %v", err)
	}

	if got := snapshot.Exporters["elasticsearch"].Sent; got != 145200 {
		t.Fatalf("expected exporter sent total 145200, got %v", got)
	}
	if got := snapshot.Exporters["elasticsearch"].SendFailed; got != 7 {
		t.Fatalf("expected exporter send_failed total 7, got %v", got)
	}
}
