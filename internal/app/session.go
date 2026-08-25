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

func (s *Session) Attach() error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}

	if s.Multiplexer == nil {
		return fmt.Errorf("multiplexer session is unavailable")
	}

	if err := multiplexer.RequireCapability(
		s.Multiplexer.Backend,
		multiplexer.CapabilityAttach,
	); err != nil {
		return err
	}

	if err := s.Multiplexer.Backend.Attach(
		s.Multiplexer.Session,
	); err != nil {
		return fmt.Errorf(
		"attach multiplexer session: %w",
		err,
	)
	}

	return nil
}

func (s *Session) Cleanup() error {
	if s == nil {
		return nil
	}

	var firstErr error

	if s.Multiplexer != nil {
		runtimePath := s.Multiplexer.Session.Runtime

		if runtimePath != "" {
			manager := multiplexer.NewManager(
				multiplexer.NewRegistry(s.Multiplexer.Backend),
			)

			if err := manager.Cleanup(runtimePath); err != nil {
				firstErr = fmt.Errorf(
					"cleanup multiplexer session: %w",
					err,
				)
			}
		} else if err := s.Multiplexer.Backend.Destroy(
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
