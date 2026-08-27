package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Resolver struct {
	Root string
}

type Result struct {
	SharedPath  string
	ProfilePath string
}

func NewResolver(root string) Resolver {
	return Resolver{
		Root: root,
	}
}

func (r Resolver) Resolve(shell, name string) (Result, error) {
	if strings.TrimSpace(shell) == "" {
		return Result{}, fmt.Errorf("shell cannot be empty")
	}

	if strings.TrimSpace(name) == "" {
		return Result{}, nil
	}

	profilePath := filepath.Join(
		r.Root,
		shell,
		fmt.Sprintf("%s.%s", name, shell),
	)

	info, err := os.Stat(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf(
				"shell profile %q not found for shell %q",
				name,
				shell,
			)
		}

		return Result{}, fmt.Errorf(
			"inspect shell profile %q for shell %q: %w",
			name,
			shell,
			err,
		)
	}

	if !info.Mode().IsRegular() {
		return Result{}, fmt.Errorf(
			"shell profile %q for shell %q is not a regular file",
			name,
			shell,
		)
	}

	return Result{
		SharedPath:  filepath.Join(r.Root, "shared", "config.toml"),
		ProfilePath: profilePath,
	}, nil
}
