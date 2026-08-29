package shell

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/mainops/uniShell/internal/shell/config"
	"gitlab.com/mainops/uniShell/internal/shell/profile"
)

func PrepareProfileStartup(
	runtimeDir string,
	shellName string,
	profileName string,
	loaded profile.Loaded,
	includeShared bool,
) (Startup, error) {
	if runtimeDir == "" {
		return Startup{}, fmt.Errorf(
			"shell startup runtime directory cannot be empty",
		)
	}

	if shellName == "" {
		return Startup{}, fmt.Errorf(
			"shell name cannot be empty",
		)
	}

	if profileName == "" {
		return Startup{}, fmt.Errorf(
			"shell profile name cannot be empty",
		)
	}

	adapter, err := config.AdapterFor(shellName)
	if err != nil {
		return Startup{}, fmt.Errorf(
			"select shell configuration adapter: %w",
			err,
		)
	}

	renderedShared := ""

	if includeShared {
		renderedShared, err = adapter.Render(loaded.Shared)
		if err != nil {
			return Startup{}, fmt.Errorf(
				"render shared shell configuration: %w",
				err,
			)
		}
	}

	startupPath := filepath.Join(
		runtimeDir,
		"config",
		"shell-generated",
		fmt.Sprintf(
			"%s.%s",
			profileName,
			shellName,
		),
	)

	if err := os.MkdirAll(
		filepath.Dir(startupPath),
		0700,
	); err != nil {
		return Startup{}, fmt.Errorf(
			"create shell startup directory: %w",
			err,
		)
	}

	data := make([]byte, 0, len(renderedShared)+len(loaded.Profile)+2)

	if renderedShared != "" {
		data = append(data, renderedShared...)
		if len(loaded.Profile) > 0 &&
			renderedShared[len(renderedShared)-1] != '\n' {
			data = append(data, '\n')
		}
	}

	data = append(data, loaded.Profile...)

	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	if err := os.WriteFile(
		startupPath,
		data,
		0600,
	); err != nil {
		return Startup{}, fmt.Errorf(
			"write shell startup file: %w",
			err,
		)
	}

	startup, err := BuildStartup(
		shellName,
		startupPath,
		runtimeDir,
	)
	if err != nil {
		return Startup{}, fmt.Errorf(
			"build shell startup: %w",
			err,
		)
	}

	return startup, nil
}

func PrepareProfileStartupFromProfile(
	runtimeDir string,
	shellName string,
	profileName string,
	includeShared bool,
) (Startup, error) {
	if runtimeDir == "" {
		return Startup{}, fmt.Errorf(
			"shell startup runtime directory cannot be empty",
		)
	}

	if shellName == "" {
		return Startup{}, fmt.Errorf(
			"shell name cannot be empty",
		)
	}

	if profileName == "" {
		return Startup{}, nil
	}

	profileRoot := filepath.Join(
		runtimeDir,
		"config",
		"shell",
	)

	loader := profile.NewLoader(profileRoot)

	loaded, err := loader.Load(
		shellName,
		profileName,
		includeShared,
	)
	if err != nil {
		return Startup{}, fmt.Errorf(
			"load shell profile %q: %w",
			profileName,
			err,
		)
	}

	startup, err := PrepareProfileStartup(
		runtimeDir,
		shellName,
		profileName,
		loaded,
		includeShared,
	)
	if err != nil {
		return Startup{}, fmt.Errorf(
			"prepare shell profile startup: %w",
			err,
		)
	}

	return startup, nil
}
