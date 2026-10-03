package server

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

func (s *Server) clock() time.Time {
	return time.Now()
}

func observeResult[T any](ctx context.Context, s *Server, method string, fn func() (T, error)) (T, error) {
	start := s.clock()
	resp, err := fn()
	s.observe(ctx, method, start, err)
	return resp, err
}

// observe records one RPC. The log carries the trace id, the running request count, and the latency.
func (s *Server) observe(ctx context.Context, method string, start time.Time, err error) {
	if s == nil {
		return
	}
	log := s.requestLogger(ctx)
	event := log.Info().
		Str("rpc", method).
		Int64("requests", s.requests.Add(1)).
		Dur("latency", time.Since(start))
	if err != nil {
		event = event.Str("error", err.Error())
	}
	event.Msg("rpc")
}

// SetLogger sets the logger used for request logs. The zero logger discards events.
func (s *Server) SetLogger(logger zerolog.Logger) {
	s.logger = logger
}

// requestLogger attaches the spec trace id when the interceptor stored one.
func (s *Server) requestLogger(ctx context.Context) zerolog.Logger {
	return pluginsdk.WithTrace(ctx, s.logger)
}
