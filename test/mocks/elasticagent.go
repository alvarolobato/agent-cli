package mocks

import (
	"encoding/json"
	"net/http"
)

// ElasticAgentHandler serves a minimal Elastic Agent status payload.
func ElasticAgentHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "ea-mock-1",
			"name": "mock-elastic-agent",
			"status": map[string]any{
				"overall": "HEALTHY",
				"message": "Running",
			},
			"components": []map[string]any{
				{
					"id":   "filestream-default",
					"name": "filestream",
					"status": map[string]any{
						"overall": "HEALTHY",
						"message": "ok",
					},
				},
				{
					"id":   "output-default",
					"name": "output-default",
					"status": map[string]any{
						"overall": "HEALTHY",
						"message": "ok",
					},
				},
				{
					"id":   "output-monitoring",
					"name": "output-monitoring",
					"status": map[string]any{
						"overall": "DEGRADED",
						"message": "slow",
					},
				},
			},
		})
	})
}
