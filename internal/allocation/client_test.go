package allocation //nolint:testpackage // Package name intentionally matches implementation for simplicity

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		BaseURL:  "http://localhost:9090",
		APIToken: "test-token",
		Timeout:  30 * time.Second,
	}

	client, err := NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.cfg.BaseURL != cfg.BaseURL {
		t.Errorf("Expected BaseURL %s, got %s", cfg.BaseURL, client.cfg.BaseURL)
	}

	if client.http == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestAllocationQuery(t *testing.T) {
	query := Query{
		Window: "30d",
		Filter: map[string]string{
			"namespace": "default",
			"pod":       "test-pod",
		},
		AggregateBy: []string{"namespace", "pod"},
	}

	if query.Window != "30d" {
		t.Errorf("Expected window 30d, got %s", query.Window)
	}

	if len(query.Filter) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(query.Filter))
	}

	if len(query.AggregateBy) != 2 {
		t.Errorf("Expected 2 aggregate fields, got %d", len(query.AggregateBy))
	}
}
