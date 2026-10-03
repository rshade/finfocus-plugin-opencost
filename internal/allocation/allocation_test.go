package allocation //nolint:testpackage // Package name intentionally matches implementation for simplicity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAllocationEntry(t *testing.T) {
	entry := AllocationEntry{
		Name: "test-allocation",
		Properties: AllocationProperties{
			Cluster:        "test-cluster",
			Node:           "test-node",
			Container:      "test-container",
			Controller:     "test-controller",
			ControllerKind: "Deployment",
			Namespace:      "default",
			Pod:            "test-pod",
			Services:       []string{"test-service"},
			Labels:         map[string]string{"app": "test"},
			Annotations:    map[string]string{"version": "v1"},
		},
		Window: AllocationWindow{
			Start: "2024-01-01T00:00:00Z",
			End:   "2024-01-01T23:59:59Z",
		},
		CPUCost:          50.25,
		GPUCost:          100.50,
		NetworkCost:      10.00,
		LoadBalancerCost: 5.00,
		PVCost:           25.00,
		RAMCost:          30.15,
		TotalCost:        220.90,
	}

	if entry.Name != "test-allocation" {
		t.Errorf("Expected name %s, got %s", "test-allocation", entry.Name)
	}

	if entry.Properties.Cluster != "test-cluster" {
		t.Errorf("Expected cluster %s, got %s", "test-cluster", entry.Properties.Cluster)
	}

	if entry.TotalCost != 220.90 {
		t.Errorf("Expected total cost %f, got %f", 220.90, entry.TotalCost)
	}

	// Verify cost calculation
	calculatedTotal := entry.CPUCost + entry.GPUCost + entry.NetworkCost +
		entry.LoadBalancerCost + entry.PVCost + entry.RAMCost
	if calculatedTotal != entry.TotalCost {
		t.Errorf("Cost components don't sum to total: %f != %f", calculatedTotal, entry.TotalCost)
	}
}

func TestAllocationProperties(t *testing.T) {
	props := AllocationProperties{
		Cluster:     "test-cluster",
		Services:    []string{"service1", "service2"},
		Labels:      map[string]string{"app": "test", "env": "prod"},
		Annotations: map[string]string{"version": "v1", "team": "platform"},
	}

	if props.Cluster != "test-cluster" {
		t.Errorf("Expected cluster %s, got %s", "test-cluster", props.Cluster)
	}

	if len(props.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(props.Services))
	}

	if len(props.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(props.Labels))
	}

	if len(props.Annotations) != 2 {
		t.Errorf("Expected 2 annotations, got %d", len(props.Annotations))
	}
}

func TestBuildAllocationURL(t *testing.T) {
	client := &Client{
		cfg: Config{
			BaseURL: "http://localhost:9090",
		},
	}

	query := AllocationQuery{
		Window: "30d",
		Filter: map[string]string{
			"namespace": "default",
			"pod":       "test-pod",
		},
		AggregateBy: []string{"namespace", "pod"},
	}

	url, err := client.BuildAllocationURL(query)
	if err != nil {
		t.Fatalf("BuildAllocationURL failed: %v", err)
	}

	expectedBase := "http://localhost:9090/model/allocation"
	if url[:len(expectedBase)] != expectedBase {
		t.Errorf("Expected URL to start with %s, got %s", expectedBase, url)
	}

	if !contains(url, "window=30d") {
		t.Error("Expected URL to contain window parameter")
	}

	if !contains(url, "filter=namespace%3A%22default%22%2Bpod%3A%22test-pod%22") &&
		!contains(url, "filter=pod%3A%22test-pod%22%2Bnamespace%3A%22default%22") {
		t.Errorf("Expected URL to contain filter parameters, got: %s", url)
	}

	if !contains(url, "aggregate=namespace%2Cpod") {
		t.Error("Expected URL to contain aggregate parameters")
	}
}

func TestBuildAllocationURL_InvalidBaseURL(t *testing.T) {
	client := &Client{
		cfg: Config{
			BaseURL: "://invalid-url",
		},
	}

	query := AllocationQuery{
		Window: "30d",
	}

	_, err := client.BuildAllocationURL(query)
	if err == nil {
		t.Error("Expected error for invalid base URL")
	}
}

func TestGetDetailedAllocation(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("Expected Authorization header")
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Error("Expected Accept header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"code": 200,
			"status": "success",
			"data": [
				{
					"test-allocation": {
						"name": "test-allocation",
						"properties": {
							"cluster": "test-cluster",
							"namespace": "default"
						},
						"window": {
							"start": "2024-01-01T00:00:00Z",
							"end": "2024-01-01T23:59:59Z"
						},
						"start": "2024-01-01T00:00:00Z",
						"end": "2024-01-01T23:59:59Z",
						"totalCost": 100.50,
						"cpuCost": 50.25,
						"ramCost": 30.15,
						"gpuCost": 10.10,
						"pvCost": 5.00,
						"networkCost": 5.00
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := &Client{
		cfg: Config{
			BaseURL:  server.URL,
			APIToken: "test-token",
			Timeout:  30 * time.Second,
		},
		http: &http.Client{},
	}

	query := AllocationQuery{
		Window: "30d",
		Filter: map[string]string{
			"namespace": "default",
		},
	}

	resp, err := client.GetDetailedAllocation(context.Background(), query)
	if err != nil {
		t.Fatalf("GetDetailedAllocation failed: %v", err)
	}

	if resp.Code != 200 {
		t.Errorf("Expected code 200, got %d", resp.Code)
	}

	if len(resp.Data) != 1 {
		t.Errorf("Expected 1 data entry, got %d", len(resp.Data))
	}

	// Check the allocation entry
	for _, dayData := range resp.Data {
		for _, entry := range dayData {
			if entry.Name != "test-allocation" {
				t.Errorf("Expected name %s, got %s", "test-allocation", entry.Name)
			}

			if entry.TotalCost != 100.50 {
				t.Errorf("Expected total cost %f, got %f", 100.50, entry.TotalCost)
			}
		}
	}
}

func TestGetDetailedAllocation_Error(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Bad request"}`))
	}))
	defer server.Close()

	client := &Client{
		cfg: Config{
			BaseURL:  server.URL,
			APIToken: "test-token",
			Timeout:  30 * time.Second,
		},
		http: &http.Client{},
	}

	query := AllocationQuery{
		Window: "30d",
	}

	_, err := client.GetDetailedAllocation(context.Background(), query)
	if err == nil {
		t.Error("Expected error for 400 status code")
	}
}

func TestConvertToSimpleResponse(t *testing.T) {
	detailed := &DetailedAllocationResponse{
		Code:   200,
		Status: "success",
		Data: []map[string]AllocationEntry{
			{
				"test-allocation": {
					Name: "test-allocation",
					Window: AllocationWindow{
						Start: "2024-01-01T00:00:00Z",
						End:   "2024-01-01T23:59:59Z",
					},
					Start:       "2024-01-01T00:00:00Z",
					End:         "2024-01-01T23:59:59Z",
					TotalCost:   100.50,
					CPUCost:     50.25,
					RAMCost:     30.15,
					GPUCost:     10.10,
					PVCost:      5.00,
					NetworkCost: 5.00,
				},
			},
		},
	}

	simple := ConvertToSimpleResponse(detailed)

	if len(simple.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(simple.Items))
	}

	item := simple.Items[0]
	if item.Cost != 100.50 {
		t.Errorf("Expected cost %f, got %f", 100.50, item.Cost)
	}

	if item.Start != "2024-01-01T00:00:00Z" {
		t.Errorf("Expected start %s, got %s", "2024-01-01T00:00:00Z", item.Start)
	}
}

func TestFormatTimeWindow(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	window := FormatTimeWindow(start, end)
	expected := "2024-01-01T00:00:00Z,2024-01-31T23:59:59Z"

	if window != expected {
		t.Errorf("Expected window %s, got %s", expected, window)
	}
}

func TestParseDurationWindow(t *testing.T) {
	// Test day format
	start, end, err := ParseDurationWindow("30d")
	if err != nil {
		t.Fatalf("ParseDurationWindow failed: %v", err)
	}

	now := time.Now().UTC()
	expectedStart := now.AddDate(0, 0, -30)

	// Allow for small time differences
	if start.Sub(expectedStart) > time.Second {
		t.Errorf("Expected start time around %v, got %v", expectedStart, start)
	}

	if end.Sub(now) > time.Second {
		t.Errorf("Expected end time around %v, got %v", now, end)
	}

	// Test standard duration
	start, endTime, err := ParseDurationWindow("24h")
	if err != nil {
		t.Fatalf("ParseDurationWindow failed: %v", err)
	}

	expectedStart = now.Add(-24 * time.Hour)
	if start.Sub(expectedStart) > time.Second {
		t.Errorf("Expected start time around %v, got %v", expectedStart, start)
	}
	_ = endTime // Use the variable to avoid unused warning
}

func TestParseDurationWindow_Invalid(t *testing.T) {
	_, _, err := ParseDurationWindow("invalid")
	if err == nil {
		t.Error("Expected error for invalid duration")
	}

	_, _, err = ParseDurationWindow("30x")
	if err == nil {
		t.Error("Expected error for invalid duration format")
	}
}

func TestEnhancedAllocation(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"code": 200,
			"status": "success",
			"data": [
				{
					"test-allocation": {
						"name": "test-allocation",
						"window": {
							"start": "2024-01-01T00:00:00Z",
							"end": "2024-01-01T23:59:59Z"
						},
						"start": "2024-01-01T00:00:00Z",
						"end": "2024-01-01T23:59:59Z",
						"totalCost": 100.50,
						"cpuCost": 50.25,
						"ramCost": 30.15,
						"gpuCost": 10.10,
						"pvCost": 5.00,
						"networkCost": 5.00
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := &Client{
		cfg: Config{
			BaseURL:  server.URL,
			APIToken: "test-token",
			Timeout:  30 * time.Second,
		},
		http: &http.Client{},
	}

	query := AllocationQuery{
		Window: "30d",
	}

	resp, err := client.EnhancedAllocation(context.Background(), query)
	if err != nil {
		t.Fatalf("EnhancedAllocation failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(resp.Items))
	}

	item := resp.Items[0]
	if item.Cost != 100.50 {
		t.Errorf("Expected cost %f, got %f", 100.50, item.Cost)
	}
}

// Helper function to check if string contains substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
