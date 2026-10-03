package allocation //nolint:testpackage // Package name intentionally matches implementation for simplicity

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestAllocationPoint(t *testing.T) {
	point := Point{
		Cost:        100.50,
		CPUCost:     50.25,
		RAMCost:     30.15,
		GPUCost:     10.10,
		PVCCost:     5.00,
		NetworkCost: 5.00,
	}

	if point.Cost != 100.50 {
		t.Errorf("Expected cost 100.50, got %f", point.Cost)
	}

	if point.CPUCost+point.RAMCost+point.GPUCost+point.PVCCost+point.NetworkCost != point.Cost {
		t.Error("Sum of individual costs should equal total cost")
	}
}

func TestClientAllocation(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("Expected Authorization header with Bearer token")
		}

		// Check URL path
		if r.URL.Path != "/model/allocation" {
			t.Errorf("Expected path /model/allocation, got %s", r.URL.Path)
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"items": [
				{
					"start": "2024-01-01T00:00:00Z",
					"end": "2024-01-01T23:59:59Z",
					"cost": 100.50,
					"cpuCost": 50.25,
					"ramCost": 30.15,
					"gpuCost": 10.10,
					"pvcCost": 5.00,
					"networkCost": 5.00
				}
			]
		}`))
	}))
	defer server.Close()

	// Create client with test server URL
	cfg := Config{
		BaseURL:  server.URL,
		APIToken: "test-token",
		Profile:  ProfileKubecost,
		Timeout:  30 * time.Second,
	}

	client, err := NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Test allocation query
	query := Query{
		Window: "30d",
		Filter: map[string]string{
			"namespace": "default",
		},
	}

	resp, err := client.Allocation(context.Background(), query)
	if err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(resp.Items))
	}

	item := resp.Items[0]
	if item.Cost != 100.50 {
		t.Errorf("Expected cost 100.50, got %f", item.Cost)
	}
}

func TestClientAllocationError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	cfg := Config{
		BaseURL:  server.URL,
		APIToken: "test-token",
		Timeout:  30 * time.Second,
	}

	client, err := NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	query := Query{
		Window: "30d",
	}

	_, err = client.Allocation(context.Background(), query)
	if err == nil {
		t.Error("Expected error for 500 status code")
	}
}

func TestClientAllocationNetworkError(t *testing.T) {
	// Create client with invalid URL to simulate network error
	cfg := Config{
		BaseURL:  "http://invalid-url-that-does-not-exist:9999",
		APIToken: "test-token",
		Timeout:  1 * time.Second, // Short timeout for test
	}

	client, err := NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	query := Query{
		Window: "30d",
	}

	_, err = client.Allocation(context.Background(), query)
	if err == nil {
		t.Error("Expected network error for invalid URL")
	}
}

func TestAllocationResponse(t *testing.T) {
	resp := Response{
		Items: []Point{
			{
				Start: "2024-01-01T00:00:00Z",
				End:   "2024-01-01T23:59:59Z",
				Cost:  100.50,
			},
			{
				Start: "2024-01-02T00:00:00Z",
				End:   "2024-01-02T23:59:59Z",
				Cost:  150.75,
			},
		},
	}

	if len(resp.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resp.Items))
	}

	totalCost := 0.0
	for _, item := range resp.Items {
		totalCost += item.Cost
	}

	expectedTotal := 100.50 + 150.75
	if totalCost != expectedTotal {
		t.Errorf("Expected total cost %f, got %f", expectedTotal, totalCost)
	}
}
