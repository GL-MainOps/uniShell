package main

import (
	"fmt"

	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

func formatCleanSessionLabel(
	metadata sessionmeta.Metadata,
) string {
	switch metadata.Mode {
	case sessionmeta.ModeNormal:
		return fmt.Sprintf(
			"direct-shell: %s",
			metadata.Name,
		)

	case sessionmeta.ModeMultiplexer:
		if metadata.Multiplexer == "" {
			return fmt.Sprintf(
				"Multiplexer: %s",
				metadata.Name,
			)
		}

		return fmt.Sprintf(
			"Multiplexer-%s: %s",
			metadata.Multiplexer,
			metadata.Name,
		)

	default:
		return fmt.Sprintf(
			"%s: %s",
			metadata.Mode,
			metadata.Name,
		)
	}
}
