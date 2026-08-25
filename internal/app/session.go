package app

import (
	"fmt"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
)

type Session struct {
	Runtime     *runtime.Session
	Multiplexer *multiplexer.ManagedSession
}

func (s *Session) Cleanup() error {
	if s == nil {
		return nil
	}

	var firstErr error

	if s.Multiplexer != nil {
		if err := s.Multiplexer.Backend.Destroy(
			s.Multiplexer.Session,
		); err != nil {
			firstErr = fmt.Errorf(
				"destroy multiplexer session: %w",
				err,
			)
		}
	}

	if s.Runtime != nil {
		if err := s.Runtime.Cleanup(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf(
				"cleanup runtime session: %w",
				err,
			)
		}
	}

	return firstErr
}
