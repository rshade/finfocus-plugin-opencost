package server

import (
	"context"
	"errors"
)

// Check probes the allocation backend. A nil error means the backend health path succeeded.
func (s *Server) Check(ctx context.Context) error {
	start := s.clock()
	var err error
	if s.cli == nil {
		err = errors.New("allocation health: nil client")
	} else {
		err = s.cli.Probe(ctx)
	}
	s.observe(ctx, "HealthCheck", start, err)
	return err
}
