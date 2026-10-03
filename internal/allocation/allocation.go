package allocation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	httpClientErrorStatus = 400
	httpSuccessStatus     = 200
)

// DetailedAllocationResponse represents the full response from Kubecost allocation API.
type DetailedAllocationResponse struct {
	Code     int                `json:"code"`
	Status   string             `json:"status"`
	Message  string             `json:"message,omitempty"`
	Currency string             `json:"currency,omitempty"`
	Data     []map[string]Entry `json:"data"`
	// FetchedUntil is when a cached copy of this body becomes stale. It is not a wire field.
	FetchedUntil time.Time `json:"-"`
}

// Entry represents a single allocation entry from Kubecost.
type Entry struct {
	Name              string                 `json:"name"`
	Properties        Properties             `json:"properties"`
	Window            Window                 `json:"window"`
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
	Currency          string                 `json:"currency,omitempty"`
	RawAllocationOnly map[string]interface{} `json:"rawAllocationOnly,omitempty"`
}

// ResponseCurrency returns the single currency named by an allocation body.
// An empty result means the body did not name one. Two different values is an error.
func ResponseCurrency(resp *DetailedAllocationResponse) (string, error) {
	if resp == nil {
		return "", nil
	}
	var found string
	consider := func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" || value == found {
			return nil
		}
		if found != "" {
			return errors.New("allocation response has more than one currency")
		}
		found = value
		return nil
	}
	if err := consider(resp.Currency); err != nil {
		return "", err
	}
	for _, step := range resp.Data {
		for _, entry := range step {
			if err := consider(entry.Currency); err != nil {
				return "", err
			}
		}
	}
	return found, nil
}

// Properties contains metadata about the allocation.
type Properties struct {
	Cluster         string            `json:"cluster,omitempty"`
	Node            string            `json:"node,omitempty"`
	Container       string            `json:"container,omitempty"`
	Controller      string            `json:"controller,omitempty"`
	ControllerKind  string            `json:"controllerKind,omitempty"`
	Namespace       string            `json:"namespace,omitempty"`
	Pod             string            `json:"pod,omitempty"`
	Services        []string          `json:"services,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	NamespaceLabels map[string]string `json:"namespaceLabels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

// Window represents the time window for the allocation.
type Window struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// BuildAllocationURL constructs the allocation URL for the configured profile.
func (c *Client) BuildAllocationURL(q Query) (string, error) {
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
	filter, err := filterValue(q.Filter)
	if err != nil {
		return "", err
	}
	if filter != "" {
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

type hostileFilterError struct{}

func (hostileFilterError) Error() string {
	return "filter value contains a reserved character"
}

// IsHostileFilter reports a filter key or value that would change the OpenCost filter grammar.
func IsHostileFilter(err error) bool {
	return errors.Is(err, hostileFilterError{})
}

func filterValue(filter map[string]string) (string, error) {
	if len(filter) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(filter))
	for key := range filter {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if err := rejectHostileFilter(key); err != nil {
			return "", err
		}
		if err := rejectHostileFilter(filter[key]); err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf(`%s:"%s"`, key, filter[key]))
	}
	return strings.Join(parts, "+"), nil
}

func rejectHostileFilter(value string) error {
	if strings.ContainsAny(value, "\"+\\()") || strings.ContainsFunc(value, unicode.IsSpace) {
		return hostileFilterError{}
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	profile, err := c.cfg.resolvedProfile()
	if err != nil || profile != ProfileKubecost || c.cfg.APIToken == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)
}

// GetDetailedAllocation retrieves detailed allocation data from Kubecost.
func (c *Client) GetDetailedAllocation(ctx context.Context, q Query) (*DetailedAllocationResponse, error) {
	endpoint, err := c.BuildAllocationURL(q)
	if err != nil {
		return nil, err
	}
	if hit, ok := c.cache.get(endpoint, c.clock()); ok {
		return allocationFromCache(hit)
	}
	if err = c.limiter.allow(c.clock()); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	decoded, err := DecodeAllocationBody(resp.StatusCode, body)
	if err != nil {
		return nil, err
	}
	decoded.FetchedUntil = c.cache.put(endpoint, resp.StatusCode, body, c.clock())
	return decoded, nil
}

func allocationFromCache(hit cachedBody) (*DetailedAllocationResponse, error) {
	decoded, err := DecodeAllocationBody(hit.status, hit.body)
	if err != nil {
		return nil, err
	}
	decoded.FetchedUntil = hit.expires
	return decoded, nil
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
func ConvertToSimpleResponse(detailed *DetailedAllocationResponse) Response {
	var items []Point

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

			items = append(items, Point{
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

	return Response{Items: items}
}

// EnhancedAllocation method that uses detailed allocation API to retrieve allocation data.
func (c *Client) EnhancedAllocation(ctx context.Context, q Query) (Response, error) {
	detailed, err := c.GetDetailedAllocation(ctx, q)
	if err != nil {
		return Response{}, err
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
