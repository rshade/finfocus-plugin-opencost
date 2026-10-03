package allocation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	httpRedirectStatus = 300
	httpClientError    = 400
)

type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(_ context.Context, cfg Config) (*Client, error) {
	return &Client{
		cfg:  cfg,
		http: &http.Client{},
	}, nil
}

// GetConfig returns the client configuration.
func (c *Client) GetConfig() Config {
	return c.cfg
}

type AllocationQuery struct {
	Window      string            // "2025-07-01T00:00:00Z,2025-07-31T23:59:59Z" or "30d"
	Filter      map[string]string // namespace, controller, pod, cluster, label:app, node, etc.
	AggregateBy []string          // e.g., ["namespace", "controller"]
}

type AllocationPoint struct {
	Start       string  `json:"start"`
	End         string  `json:"end"`
	Cost        float64 `json:"cost"`
	CPUCost     float64 `json:"cpuCost"`
	RAMCost     float64 `json:"ramCost"`
	GPUCost     float64 `json:"gpuCost"`
	PVCCost     float64 `json:"pvcCost"`
	NetworkCost float64 `json:"networkCost"`
	// ... add fields as needed
}

type AllocationResponse struct {
	Items []AllocationPoint `json:"items"`
}

// PredictionRequest represents the request for cost prediction API.
type PredictionRequest struct {
	ClusterID        string `json:"clusterID"`
	DefaultNamespace string `json:"defaultNamespace"`
	Window           string `json:"window,omitempty"`  // Optional: duration for cost prediction (default: "2d")
	NoUsage          bool   `json:"noUsage,omitempty"` // Optional: ignore historical usage data
	WorkloadSpec     string `json:"workloadSpec"`      // YAML or JSON workload specification
}

// PredictionResponse represents the response from cost prediction API.
type PredictionResponse struct {
	CostBefore string `json:"costBefore"` // Current monthly cost
	CostAfter  string `json:"costAfter"`  // Projected monthly cost
	CostChange string `json:"costChange"` // Difference between costs
}

func (c *Client) Allocation(ctx context.Context, q AllocationQuery) (AllocationResponse, error) {
	url, err := c.BuildAllocationURL(q)
	if err != nil {
		return AllocationResponse{}, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	c.setAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return AllocationResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= httpRedirectStatus {
		return AllocationResponse{}, fmt.Errorf("kubecost %d", resp.StatusCode)
	}
	var out AllocationResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&out); decodeErr != nil {
		return out, decodeErr
	}
	return out, nil
}

// PredictSpecCost sends a workload specification to the Kubecost prediction API
// and returns the predicted cost impact.
func (c *Client) PredictSpecCost(ctx context.Context, req PredictionRequest) (PredictionResponse, error) {
	// Build the prediction API URL
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return PredictionResponse{}, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = "/model/prediction/speccost"

	// Build query parameters
	params := url.Values{}
	params.Set("clusterID", req.ClusterID)
	params.Set("defaultNamespace", req.DefaultNamespace)

	// Add optional parameters
	if req.Window != "" {
		params.Set("window", req.Window)
	}
	if req.NoUsage {
		params.Set("noUsage", "true")
	}

	u.RawQuery = params.Encode()

	// Determine content type based on workload spec format
	contentType := "application/yaml"
	if isJSON(req.WorkloadSpec) {
		contentType = "application/json"
	}

	// Create the HTTP request with workload spec as body
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u.String(),
		bytes.NewBufferString(req.WorkloadSpec),
	)
	if err != nil {
		return PredictionResponse{}, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Accept", "application/json")
	if c.cfg.APIToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)
	}

	// Execute the request
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return PredictionResponse{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode >= httpClientError {
		return PredictionResponse{}, fmt.Errorf("kubecost prediction API error: status=%d", resp.StatusCode)
	}

	// Decode the response
	var predResp PredictionResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&predResp); decodeErr != nil {
		return PredictionResponse{}, fmt.Errorf("decoding response: %w", decodeErr)
	}

	return predResp, nil
}

// isJSON checks if the string appears to be JSON format.
func isJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}
