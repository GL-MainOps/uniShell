package profile

import (
	"fmt"
	"os"

	"gitlab.com/mainops/uniShell/internal/shell/config"
)

type Loaded struct {
	Shared  config.Config
	Profile []byte
}

type Loader struct {
	Resolver Resolver
}

func NewLoader(root string) Loader {
	return Loader{
		Resolver: NewResolver(root),
	}
}

func (l Loader) Load(
	shell string,
	name string,
	includeShared bool,
) (Loaded, error) {
	resolved, err := l.Resolver.Resolve(shell, name)
	if err != nil {
		return Loaded{}, err
	}

	var loaded Loaded

	if includeShared {
		data, err := os.ReadFile(resolved.SharedPath)
		if err != nil {
			if os.IsNotExist(err) {
				return Loaded{}, fmt.Errorf(
					"shared shell configuration not found: %s",
					resolved.SharedPath,
				)
			}

			return Loaded{}, fmt.Errorf(
				"read shared shell configuration: %w",
				err,
			)
		}

		shared, err := config.Parse(data)
		if err != nil {
			return Loaded{}, fmt.Errorf(
				"parse shared shell configuration: %w",
				err,
			)
		}

		loaded.Shared = shared
	}

	if name == "" {
		return loaded, nil
	}

	data, err := os.ReadFile(resolved.ProfilePath)
	if err != nil {
		return Loaded{}, fmt.Errorf(
			"read shell profile %q: %w",
			name,
			err,
		)
	}

	loaded.Profile = data

	return loaded, nil

}
