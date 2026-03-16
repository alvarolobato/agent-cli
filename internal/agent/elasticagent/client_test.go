package elasticagent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientGetStatusAndComponents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			t.Fatalf("expected path /api/status, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"agent-1",
			"name":"my-agent",
			"status":{"overall":"HEALTHY","message":"Running"},
			"components":[{"id":"filestream-default","name":"filestream","status":{"overall":"HEALTHY","message":"Running"}}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())

	status, err := client.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.Status.Overall != "HEALTHY" {
		t.Fatalf("expected overall HEALTHY, got %q", status.Status.Overall)
	}
	if len(status.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(status.Components))
	}

	components, err := client.GetComponents(context.Background())
	if err != nil {
		t.Fatalf("GetComponents() error = %v", err)
	}
	if len(components) != 1 {
		t.Fatalf("expected 1 component from GetComponents, got %d", len(components))
	}
	if components[0].ID != "filestream-default" {
		t.Fatalf("expected component id filestream-default, got %q", components[0].ID)
	}
}

func TestClientGetStatusConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", &http.Client{})
	_, err := client.GetStatus(context.Background())
	if err == nil {
		t.Fatalf("expected connection error, got nil")
	}
	if !strings.Contains(err.Error(), "request elastic agent status") {
		t.Fatalf("expected wrapped request error, got %q", err.Error())
	}
}

func TestClientGetStatusFallsBackToStatsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			http.NotFound(w, r)
		case "/stats":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"beat":{
					"info":{
						"ephemeral_id":"ephemeral-1",
						"name":"elastic-agent"
					}
				}
			}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	status, err := client.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.ID != "ephemeral-1" {
		t.Fatalf("expected fallback id ephemeral-1, got %q", status.ID)
	}
	if status.Name != "elastic-agent" {
		t.Fatalf("expected fallback name elastic-agent, got %q", status.Name)
	}
	if status.Status.Overall != "HEALTHY" {
		t.Fatalf("expected fallback health HEALTHY, got %q", status.Status.Overall)
	}
}
