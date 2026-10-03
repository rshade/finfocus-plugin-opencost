package allocation

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	httpClientErrorStatus = 400
	httpSuccessStatus     = 200
)

// DetailedAllocationResponse represents the full response from Kubecost allocation API.
type DetailedAllocationResponse struct {
	Code    int                          `json:"code"`
	Status  string                       `json:"status"`
	Message string                       `json:"message,omitempty"`
	Data    []map[string]AllocationEntry `json:"data"`
}

// AllocationEntry represents a single allocation entry from Kubecost.
type AllocationEntry struct {
	Name              string                 `json:"name"`
	Properties        AllocationProperties   `json:"properties"`
	Window            AllocationWindow       `json:"window"`
	Start             string                 `json:"start"`
	End               string                 `json:"end"`
	Minutes           float64                `json:"minutes"`
	CPUCores          float64                `json:"cpuCores"`
	CPUCoreHours      float64                `json:"cpuCoreHours"`
	CPUCost           float64                `json:"cpuCost"`
	CPUEfficiency     float64                `json:"cpuEfficiency"`
	GPUCount          float64                `json:"gpuCount"`
	GPUHours          float64                `json:"gpuHours"`
	GPUCost           float64                `json:"gpuCost"`
	NetworkCost       float64                `json:"networkCost"`
	LoadBalancerCost  float64                `json:"loadBalancerCost"`
	PVCost            float64                `json:"pvCost"`
	RAMBytes          float64                `json:"ramBytes"`
	RAMByteHours      float64                `json:"ramByteHours"`
	RAMCost           float64                `json:"ramCost"`
	RAMEfficiency     float64                `json:"ramEfficiency"`
	SharedCost        float64                `json:"sharedCost"`
	ExternalCost      float64                `json:"externalCost"`
	TotalCost         float64                `json:"totalCost"`
	TotalEfficiency   float64                `json:"totalEfficiency"`
	RawAllocationOnly map[string]interface{} `json:"rawAllocationOnly,omitempty"`
}

// AllocationProperties contains metadata about the allocation.
type AllocationProperties struct {
	Cluster        string            `json:"cluster,omitempty"`
	Node           string            `json:"node,omitempty"`
	Container      string            `json:"container,omitempty"`
	Controller     string            `json:"controller,omitempty"`
	ControllerKind string            `json:"controllerKind,omitempty"`
	Namespace      string            `json:"namespace,omitempty"`
	Pod            string            `json:"pod,omitempty"`
	Services       []string          `json:"services,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
}

// AllocationWindow represents the time window for the allocation.
type AllocationWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// BuildAllocationURL constructs the allocation URL for the configured profile.
func (c *Client) BuildAllocationURL(q AllocationQuery) (string, error) {
	profile, err := c.cfg.resolvedProfile()
	if err != nil {
		return "", err
	}
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	params := url.Values{}
	params.Set("window", q.Window)
	if filter := filterValue(q.Filter); filter != "" {
		params.Set("filter", filter)
	}
	if len(q.AggregateBy) > 0 {
		params.Set("aggregate", strings.Join(q.AggregateBy, ","))
	}
	switch profile {
	case ProfileOpenCost:
		u.Path = "/allocation"
		params.Set("includeIdle", "false")
		params.Set("shareIdle", "false")
	case ProfileKubecost:
		u.Path = "/model/allocation"
		params.Set("accumulate", "false")
		params.Set("idle", "false")
		params.Set("shareIdle", "false")
	default:
		return "", fmt.Errorf("unknown allocation profile %q", profile)
	}
	u.RawQuery = params.Encode()
	return u.String(), nil
}

func filterValue(filter map[string]string) string {
	if len(filter) == 0 {
		return ""
	}
	keys := make([]string, 0, len(filter))
	for key := range filter {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s:"%s"`, key, filter[key]))
	}
	return strings.Join(parts, "+")
}

func (c *Client) setAuth(req *http.Request) {
	profile, err := c.cfg.resolvedProfile()
	if err != nil || profile != ProfileKubecost || c.cfg.APIToken == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)
}

// GetDetailedAllocation retrieves detailed allocation data from Kubecost.
func (c *Client) GetDetailedAllocation(ctx context.Context, q AllocationQuery) (*DetailedAllocationResponse, error) {
	url, err := c.BuildAllocationURL(q)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

	// Configure HTTP client with TLS settings
	if c.http == nil {
		c.http = &http.Client{
			Timeout: c.cfg.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: c.cfg.TLSSkipVerify, //nolint:gosec // Configurable for dev environments
				},
			},
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return DecodeAllocationBody(resp.StatusCode, body)
}

// DecodeAllocationBody decodes an allocation envelope.
// HTTP errors and plain-text bodies are returned as errors that include the body.
func DecodeAllocationBody(statusCode int, body []byte) (*DetailedAllocationResponse, error) {
	trimmed := bytes.TrimSpace(body)
	if statusCode >= httpClientErrorStatus || (len(trimmed) > 0 && trimmed[0] != '{') {
		return nil, fmt.Errorf("allocation API error: status=%d, body=%s", statusCode, string(trimmed))
	}
	var result DetailedAllocationResponse
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if result.Code != httpSuccessStatus {
		return nil, fmt.Errorf("allocation API returned error code %d: %s", result.Code, result.Message)
	}
	return &result, nil
}

// ConvertToSimpleResponse converts detailed allocation to the simple response format.
func ConvertToSimpleResponse(detailed *DetailedAllocationResponse) AllocationResponse {
	var items []AllocationPoint

	for _, dayData := range detailed.Data {
		for _, entry := range dayData {
			// Parse the window times
			start := entry.Start
			end := entry.End
			if start == "" && entry.Window.Start != "" {
				start = entry.Window.Start
			}
			if end == "" && entry.Window.End != "" {
				end = entry.Window.End
			}

			items = append(items, AllocationPoint{
				Start:       start,
				End:         end,
				Cost:        entry.TotalCost,
				CPUCost:     entry.CPUCost,
				RAMCost:     entry.RAMCost,
				GPUCost:     entry.GPUCost,
				PVCCost:     entry.PVCost,
				NetworkCost: entry.NetworkCost,
			})
		}
	}

	return AllocationResponse{Items: items}
}

// EnhancedAllocation method that uses detailed allocation API to retrieve allocation data.
func (c *Client) EnhancedAllocation(ctx context.Context, q AllocationQuery) (AllocationResponse, error) {
	detailed, err := c.GetDetailedAllocation(ctx, q)
	if err != nil {
		return AllocationResponse{}, err
	}

	return ConvertToSimpleResponse(detailed), nil
}

// FormatTimeWindow formats time window for Kubecost API.
func FormatTimeWindow(start, end time.Time) string {
	// Kubecost accepts various formats, RFC3339 is most reliable
	return fmt.Sprintf("%s,%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
}

// ParseDurationWindow parses duration window (e.g., "30d", "7d", "24h").
func ParseDurationWindow(window string) (time.Time, time.Time, error) {
	now := time.Now().UTC()

	// Parse duration strings like "30d", "7d", "24h"
	if strings.HasSuffix(window, "d") {
		days := strings.TrimSuffix(window, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", window)
		}
		start := now.AddDate(0, 0, -d)
		return start, now, nil
	}

	// Try parsing as standard duration
	duration, err := time.ParseDuration(window)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %s", window)
	}

	start := now.Add(-duration)
	return start, now, nil
}
