package allocation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Probe requests the backend health path. A status of 400 or higher is unhealthy.
// The result is not cached and does not use the allocation rate limit.
func (c *Client) Probe(ctx context.Context) error {
	if c == nil {
		return errors.New("allocation health: nil client")
	}
	base, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("allocation health: %w", err)
	}
	base.Path = "/healthz"
	base.RawQuery = ""
	base.Fragment = ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return fmt.Errorf("allocation health: %w", err)
	}
	c.setAuth(req)
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("allocation health: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= httpClientError {
		return fmt.Errorf("allocation health: status=%d", resp.StatusCode)
	}
	return nil
}
