package mocks

import (
	"encoding/json"
	"net/http"
)

// ElasticAgentHandler serves a minimal Elastic Agent status payload.
func ElasticAgentHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": map[string]any{
				"overall": "HEALTHY",
				"message": "Running",
			},
			"components": []map[string]any{
				{"id": "filestream-default", "name": "filestream"},
			},
		})
	})
}
