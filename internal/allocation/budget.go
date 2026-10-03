package allocation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// BudgetRule is one row from GET /model/budgets.
// Fields follow the Kubecost Budget API response documented at
// https://www.ibm.com/docs/en/SSW0JQG_1.x/apis/apis-overview/budget-api.html
// Not verified against live Kubecost.
type BudgetRule struct {
	Name         string              `json:"name"`
	ID           string              `json:"id"`
	Values       map[string][]string `json:"values"`
	Kind         string              `json:"kind"`
	Interval     string              `json:"interval"`
	IntervalDay  int                 `json:"intervalDay"`
	SpendLimit   float64             `json:"spendLimit"`
	Actions      []BudgetAction      `json:"actions"`
	Window       BudgetWindow        `json:"window"`
	CurrentSpend float64             `json:"currentSpend"`
}

// BudgetAction is one alert threshold on a budget rule.
type BudgetAction struct {
	Amount          float64  `json:"amount"`
	Percentage      float64  `json:"percentage"`
	SlackWebhooks   []string `json:"slackWebhooks"`
	MSTeamsWebhooks []string `json:"msTeamsWebhooks"`
	Emails          []string `json:"emails"`
	LastFired       string   `json:"lastFired"`
}

// BudgetWindow is the current budget window from the list response.
type BudgetWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// ListBudgets fetches recurring budget rules from GET /model/budgets.
func (c *Client) ListBudgets(ctx context.Context) ([]BudgetRule, error) {
	if err := c.limiter.allow(c.clock()); err != nil {
		return nil, err
	}
	u, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = "/model/budgets"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuth(req)
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= httpClientError {
		return nil, fmt.Errorf("budget API error: status=%d", resp.StatusCode)
	}
	var listed struct {
		Code int          `json:"code"`
		Data []BudgetRule `json:"data"`
	}
	if decodeErr := json.Unmarshal(body, &listed); decodeErr != nil {
		return nil, fmt.Errorf("decoding response: %w", decodeErr)
	}
	if listed.Code >= httpClientError {
		return nil, fmt.Errorf("budget API error: status=%d", listed.Code)
	}
	return listed.Data, nil
}
