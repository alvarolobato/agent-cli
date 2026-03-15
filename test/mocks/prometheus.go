package mocks

import "net/http"

// PrometheusHandler serves mock OTel collector metrics.
func PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte("otelcol_receiver_accepted_spans{receiver=\"otlp\"} 123\n"))
	})
}
