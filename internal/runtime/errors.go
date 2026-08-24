package runtime

import "fmt"

// PermissionError indicates that uniShell could not access or modify
// a filesystem location because of insufficient permissions.
type PermissionError struct {
	Path   string
	Action string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf(
		"permission denied while trying to %s %q",
		e.Action,
		e.Path,
	)
}
