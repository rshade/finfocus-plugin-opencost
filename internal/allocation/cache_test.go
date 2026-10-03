package allocation //nolint:testpackage // cache clock and transport fields are unexported

import (
	"context"
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
	query := AllocationQuery{Window: "60m"}
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
	query := AllocationQuery{Window: "60m"}

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
		if _, err = client.GetDetailedAllocation(context.Background(), AllocationQuery{Window: window}); err != nil {
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
