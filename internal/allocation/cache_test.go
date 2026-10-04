package allocation //nolint:testpackage // cache clock and transport fields are unexported

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkCachedAllocation(b *testing.B) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	b.Cleanup(backend.Close)

	client, err := NewClient(context.Background(), Config{BaseURL: backend.URL})
	if err != nil {
		b.Fatal(err)
	}
	query := Query{Window: "60m"}
	if _, err = client.GetDetailedAllocation(context.Background(), query); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		if _, err = client.GetDetailedAllocation(context.Background(), query); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCachePutDropsExpiredEntries(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	cache := newResponseCache(time.Second)
	cache.put("https://opencost/a", 200, []byte("body-a"), now)
	cache.put("https://opencost/b", 200, []byte("body-b"), now)
	now = now.Add(2 * time.Second)
	cache.put("https://opencost/c", 200, []byte("body-c"), now)
	if _, ok := cache.entries["https://opencost/a"]; ok {
		t.Fatal("expired entry a is still stored")
	}
	if _, ok := cache.entries["https://opencost/b"]; ok {
		t.Fatal("expired entry b is still stored")
	}
	got, ok := cache.entries["https://opencost/c"]
	if !ok || string(got.body) != "body-c" {
		t.Fatal("live entry c was dropped")
	}
}

func TestCachePutCapsLiveEntries(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	cache := newResponseCache(time.Minute)
	for i := range maxCacheEntries + 1 {
		cache.put(fmt.Sprintf("https://opencost/%d", i), 200, []byte("x"), now.Add(time.Duration(i)))
	}
	if len(cache.entries) != maxCacheEntries {
		t.Fatalf("entries = %d, want %d", len(cache.entries), maxCacheEntries)
	}
	if _, ok := cache.entries["https://opencost/0"]; ok {
		t.Fatal("oldest live entry was kept past the cap")
	}
}

func TestCachePutKeepsARecentlyReadEntry(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	cache := newResponseCache(time.Hour)
	cache.put("https://opencost/old", 200, []byte("old"), now)
	for i := 1; i < maxCacheEntries; i++ {
		cache.put(fmt.Sprintf("https://opencost/%d", i), 200, []byte("x"), now.Add(time.Duration(i)))
	}
	readAt := now.Add(time.Minute)
	if _, ok := cache.get("https://opencost/old", readAt); !ok {
		t.Fatal("old entry was missing before the extra insert")
	}
	cache.put("https://opencost/new", 200, []byte("new"), readAt)
	if _, ok := cache.entries["https://opencost/old"]; !ok {
		t.Fatal("recently read entry was evicted")
	}
	if _, ok := cache.entries["https://opencost/1"]; ok {
		t.Fatal("least recently used entry was kept")
	}
	if _, ok := cache.entries["https://opencost/new"]; !ok {
		t.Fatal("new entry was evicted")
	}
}

func TestCachePutKeepsNewestWhenExpiriesTie(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	cache := newResponseCache(time.Minute)
	newest := fmt.Sprintf("https://opencost/%d", maxCacheEntries)
	for i := range maxCacheEntries + 1 {
		cache.put(fmt.Sprintf("https://opencost/%d", i), 200, []byte("x"), now)
	}
	if len(cache.entries) != maxCacheEntries {
		t.Fatalf("entries = %d, want %d", len(cache.entries), maxCacheEntries)
	}
	if _, ok := cache.entries[newest]; !ok {
		t.Fatal("newest entry was dropped when expiries tie")
	}
}

func TestAllocationCacheExpires(t *testing.T) {
	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	t.Cleanup(backend.Close)

	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	client, err := NewClient(context.Background(), Config{BaseURL: backend.URL, CacheTTL: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	client.now = func() time.Time { return now }
	query := Query{Window: "60m"}

	if _, err = client.GetDetailedAllocation(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	if _, err = client.GetDetailedAllocation(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls inside TTL = %d, want 1", calls.Load())
	}

	now = now.Add(2 * time.Second)
	if _, err = client.GetDetailedAllocation(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls after TTL = %d, want 2", calls.Load())
	}
}

func TestPooledClientReusesOneConnection(t *testing.T) {
	var calls atomic.Int32
	var newConns atomic.Int32
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			newConns.Add(1)
		}
	}
	backend.Start()
	t.Cleanup(backend.Close)

	client, err := NewClient(context.Background(), Config{BaseURL: backend.URL, CacheTTL: -time.Nanosecond})
	if err != nil {
		t.Fatal(err)
	}
	for _, window := range []string{"60m", "30d"} {
		if _, err = client.GetDetailedAllocation(context.Background(), Query{Window: window}); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("backend calls = %d, want 2", calls.Load())
	}
	if newConns.Load() != 1 {
		t.Fatalf("connections = %d, want 1", newConns.Load())
	}

	transport, ok := client.http.Transport.(*http.Transport)
	if !ok {
		t.Fatal("transport is not *http.Transport")
	}
	if transport.MaxIdleConnsPerHost < 2 || transport.IdleConnTimeout <= 0 || transport.DialContext == nil ||
		transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 || transport.DisableKeepAlives {
		t.Fatalf("pool is incomplete: idlePerHost=%d idle=%s dial=%v handshake=%s header=%s disableKeepAlive=%v",
			transport.MaxIdleConnsPerHost, transport.IdleConnTimeout, transport.DialContext != nil,
			transport.TLSHandshakeTimeout, transport.ResponseHeaderTimeout, transport.DisableKeepAlives)
	}
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS verification is not on")
	}
}
