package allocation

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

const (
	httpRedirectStatus   = 300
	httpClientError      = 400
	pooledIdleConns      = 10
	dialTimeout          = 5 * time.Second
	keepAlivePeriod      = 30 * time.Second
	idleConnTimeout      = 90 * time.Second
	tlsHandshakeTimeout  = 10 * time.Second
	expectContinuePeriod = time.Second
)

type Client struct {
	cfg     Config
	http    *http.Client
	logger  zerolog.Logger
	now     func() time.Time
	cache   *responseCache
	limiter *limiter
}

func NewClient(_ context.Context, cfg Config) (*Client, error) {
	return &Client{
		cfg:     cfg,
		http:    httpClient(cfg),
		now:     time.Now,
		cache:   newResponseCache(resolvedCacheTTL(cfg.CacheTTL)),
		limiter: newLimiter(cfg.RequestsPerSecond, cfg.RateBurst),
	}, nil
}

func (c *Client) clock() time.Time {
	if c != nil && c.now != nil {
		return c.now()
	}
	return time.Now()
}

// SetLogger sets the logger for outbound requests. The zero logger discards events.
// Request logs include the URL and not the API token.
func (c *Client) SetLogger(logger zerolog.Logger) {
	if c == nil {
		return
	}
	c.logger = logger
}

func httpClient(cfg Config) *http.Client {
	headerTimeout := cfg.Timeout
	if headerTimeout <= 0 {
		headerTimeout = defaultTimeoutDuration
	}
	dialer := &net.Dialer{Timeout: dialTimeout, KeepAlive: keepAlivePeriod}
	return &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          pooledIdleConns,
			MaxIdleConnsPerHost:   pooledIdleConns,
			IdleConnTimeout:       idleConnTimeout,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			ResponseHeaderTimeout: headerTimeout,
			ExpectContinueTimeout: expectContinuePeriod,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.TLSSkipVerify, //nolint:gosec // explicit opt-in, off by default
			},
		},
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if id := pluginsdk.TraceIDFromContext(req.Context()); id != "" {
		req.Header.Set(pluginsdk.TraceIDMetadataKey, id)
	}
	c.logger.Info().
		Str("method", req.Method).
		Str("url", req.URL.Redacted()).
		Msg("allocation request")
	return c.http.Do(req)
}

// GetConfig returns the client configuration.
func (c *Client) GetConfig() Config {
	return c.cfg
}

type Query struct {
	Window      string            // "2025-07-01T00:00:00Z,2025-07-31T23:59:59Z" or "30d"
	Filter      map[string]string // namespace, controller, pod, cluster, label:app, node, etc.
	AggregateBy []string          // e.g., ["namespace", "controller"]
}

type Point struct {
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

type Response struct {
	Items []Point `json:"items"`
}

// SpecCostWindowResourceCost is the kubectl-cost --window-cost default.
// pkg/cmd/predict.go at 1f45d3085b2ffa84758bfa8131ea8b7784cd8ed1.
const SpecCostWindowResourceCost = "7d offset 48h"

const specCostWindowUsageDefault = "2d"

// PredictionRequest is the POST /model/prediction/speccost query.
// Query names follow pkg/cmd/predict.go at the commit cited above.
type PredictionRequest struct {
	ClusterID          string
	DefaultNamespace   string
	WindowAvgUsage     string
	WindowResourceCost string
	NoUsage            bool
	WorkloadSpec       string
}

// CostPrediction is one side of a spec cost diff.
// json tags match pkg/query/prediction_speccost.go at that commit.
type CostPrediction struct {
	TotalMonthlyRate    float64 `json:"totalMonthlyRate"`
	CPUMonthlyRate      float64 `json:"cpuMonthlyRate"`
	RAMMonthlyRate      float64 `json:"ramMonthlyRate"`
	GPUMonthlyRate      float64 `json:"gpuMonthlyRate"`
	MonthlyCPUCoreHours float64 `json:"monthlyCPUCoreHours"`
	MonthlyRAMByteHours float64 `json:"monthlyRAMByteHours"`
	MonthlyGPUHours     float64 `json:"monthlyGPUHours"`
}

// SpecCostDiff is one predicted workload from the spec cost API.
type SpecCostDiff struct {
	Namespace      string         `json:"namespace"`
	ControllerKind string         `json:"controllerKind"`
	ControllerName string         `json:"controllerName"`
	CostBefore     CostPrediction `json:"costBefore"`
	CostAfter      CostPrediction `json:"costAfter"`
	CostChange     CostPrediction `json:"costChange"`
}

func (c *Client) Allocation(ctx context.Context, q Query) (Response, error) {
	url, err := c.BuildAllocationURL(q)
	if err != nil {
		return Response{}, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	c.setAuth(req)
	resp, err := c.do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= httpRedirectStatus {
		return Response{}, &StatusError{
			Code: resp.StatusCode,
			Msg:  fmt.Sprintf("kubecost %d", resp.StatusCode),
		}
	}
	var out Response
	if decodeErr := json.NewDecoder(resp.Body).Decode(&out); decodeErr != nil {
		return out, decodeErr
	}
	return out, nil
}

// PredictSpecCost posts a workload spec to /model/prediction/speccost.
// The response shape is the kubectl-cost SpecCostResponse array.
// Not verified against live Kubecost.
func (c *Client) PredictSpecCost(ctx context.Context, req PredictionRequest) ([]SpecCostDiff, error) {
	if err := c.limiter.allow(c.clock()); err != nil {
		return nil, err
	}
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = "/model/prediction/speccost"
	usage := req.WindowAvgUsage
	if usage == "" {
		usage = specCostWindowUsageDefault
	}
	costWindow := req.WindowResourceCost
	if costWindow == "" {
		costWindow = SpecCostWindowResourceCost
	}
	params := url.Values{}
	params.Set("clusterID", req.ClusterID)
	params.Set("defaultNamespace", req.DefaultNamespace)
	params.Set("windowAvgUsage", usage)
	params.Set("windowResourceCost", costWindow)
	params.Set("noUsage", strconv.FormatBool(req.NoUsage))
	u.RawQuery = params.Encode()

	contentType := "application/yaml"
	if isJSON(req.WorkloadSpec) {
		contentType = "application/json"
	}
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u.String(),
		bytes.NewBufferString(req.WorkloadSpec),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Accept", "application/json")
	c.setAuth(httpReq)

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= httpClientError {
		return nil, &StatusError{
			Code: resp.StatusCode,
			Msg:  fmt.Sprintf("prediction API error: status=%d", resp.StatusCode),
		}
	}
	var rows []SpecCostDiff
	if decodeErr := json.NewDecoder(resp.Body).Decode(&rows); decodeErr != nil {
		return nil, fmt.Errorf("decoding response: %w", decodeErr)
	}
	return rows, nil
}

// isJSON checks if the string appears to be JSON format.
func isJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}
