package allocation

import (
	"sync"
	"time"
)

const (
	defaultCacheTTL = 30 * time.Second
	maxCacheEntries = 256
)

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
	for existing, entry := range c.entries {
		if !now.Before(entry.expires) {
			delete(c.entries, existing)
		}
	}
	c.entries[key] = cachedBody{status: status, body: copied, expires: expires}
	for len(c.entries) > maxCacheEntries {
		oldestKey := ""
		var oldest time.Time
		for existing, entry := range c.entries {
			if existing == key {
				continue
			}
			if oldestKey == "" || entry.expires.Before(oldest) {
				oldestKey = existing
				oldest = entry.expires
			}
		}
		if oldestKey == "" {
			break
		}
		delete(c.entries, oldestKey)
	}
	c.mu.Unlock()
	return expires
}
