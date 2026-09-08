package acquisition

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func LoadTools(dir string) ([]Tool, error) {
	if dir == "" {
		return nil, fmt.Errorf("tool metadata directory is required")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf(
			"read tool metadata directory %q: %w",
			dir,
			err,
		)
	}

	paths := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".toml" {
			continue
		}

		paths = append(
			paths,
			filepath.Join(dir, entry.Name()),
		)
	}

	sort.Strings(paths)

	if len(paths) == 0 {
		return nil, fmt.Errorf(
			"tool metadata directory %q contains no TOML definitions",
			dir,
		)
	}

	tools := make([]Tool, 0)
	seen := make(map[string]string)

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf(
				"open tool metadata %q: %w",
				path,
				err,
			)
		}

		manifest, err := LoadManifest(file)
		closeErr := file.Close()

		if err != nil {
			return nil, fmt.Errorf(
				"load tool metadata %q: %w",
				path,
				err,
			)
		}

		if closeErr != nil {
			return nil, fmt.Errorf(
				"close tool metadata %q: %w",
				path,
				closeErr,
			)
		}

		fileTools, err := manifest.BuildTools()
		if err != nil {
			return nil, fmt.Errorf(
				"build tools from metadata %q: %w",
				path,
				err,
			)
		}

		for _, tool := range fileTools {
			if previousPath, exists := seen[tool.Name]; exists {
				return nil, fmt.Errorf(
					"duplicate tool %q in %q and %q",
					tool.Name,
					previousPath,
					path,
				)
			}

			seen[tool.Name] = path
			tools = append(tools, tool)
		}
	}

	return tools, nil
}
