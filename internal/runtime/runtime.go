package runtime

import (
	"errors"
	"fmt"
	"os"
)

// Prepare creates the directories required by the runtime.
func (p Paths) Prepare() error {
	directories := []struct {
		path   string
		action string
	}{
		{p.Root, "create runtime root"},
		{p.Runtime, "create runtime directory"},
		{p.Bin, "create binary directory"},
		{p.Config, "create configuration directory"},
	}

	for _, directory := range directories {
		if err := os.MkdirAll(directory.path, 0700); err != nil {
			if errors.Is(err, os.ErrPermission) {
				return &PermissionError{
					Path:   directory.path,
					Action: directory.action,
				}
			}

			return fmt.Errorf(
				"%s %q: %w",
				directory.action,
				directory.path,
				err,
			)
		}
	}

	return nil
}
