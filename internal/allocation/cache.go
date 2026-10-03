package allocation

import (
	"sync"
	"time"
)

const defaultCacheTTL = 30 * time.Second

func resolvedCacheTTL(configured time.Duration) time.Duration {
	if configured < 0 {
		return 0
	}
	if configured == 0 {
		return defaultCacheTTL
	}
	return configured
}

type cachedBody struct {
	status  int
	body    []byte
	expires time.Time
}

type responseCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]cachedBody
}

func newResponseCache(ttl time.Duration) *responseCache {
	return &responseCache{ttl: ttl, entries: make(map[string]cachedBody)}
}

func (c *responseCache) get(key string, now time.Time) (cachedBody, bool) {
	if c == nil || c.ttl <= 0 || key == "" {
		return cachedBody{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || !now.Before(entry.expires) {
		delete(c.entries, key)
		return cachedBody{}, false
	}
	return entry, true
}

func (c *responseCache) put(key string, status int, body []byte, now time.Time) time.Time {
	if c == nil || c.ttl <= 0 || key == "" {
		return time.Time{}
	}
	expires := now.Add(c.ttl)
	copied := make([]byte, len(body))
	copy(copied, body)
	c.mu.Lock()
	c.entries[key] = cachedBody{status: status, body: copied, expires: expires}
	c.mu.Unlock()
	return expires
}
