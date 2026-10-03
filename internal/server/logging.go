package server

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

// SetLogger sets the logger used for request logs. The zero logger discards events.
func (s *Server) SetLogger(logger zerolog.Logger) {
	s.logger = logger
}

// requestLogger attaches the spec trace id when the interceptor stored one.
func (s *Server) requestLogger(ctx context.Context) zerolog.Logger {
	return pluginsdk.WithTrace(ctx, s.logger)
}
