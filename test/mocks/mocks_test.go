package mocks

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestElasticAgentMockServer(t *testing.T) {
	server := httptest.NewServer(ElasticAgentHandler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatalf("Body.Close() error = %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if !strings.Contains(string(body), "\"overall\":\"HEALTHY\"") {
		t.Fatalf("expected HEALTHY payload, got: %s", string(body))
	}
}
