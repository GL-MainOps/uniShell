package config

import (
	"fmt"
	"sort"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Environment map[string]string `toml:"environment"`
	Path        PathConfig        `toml:"path"`
	Aliases     map[string]string `toml:"aliases"`
}

type PathConfig struct {
	Add []string `toml:"add"`
}

func Parse(data []byte) (Config, error) {
	var cfg Config

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf(
			"parse shell configuration: %w",
			err,
		)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	for name := range c.Environment {
		if !validIdentifier(name) {
			return fmt.Errorf(
				"invalid environment variable name %q",
				name,
			)
		}
	}

	for name := range c.Aliases {
		if !validIdentifier(name) {
			return fmt.Errorf(
				"invalid alias name %q",
				name,
			)
		}
	}

	for index, entry := range c.Path.Add {
		if entry == "" {
			return fmt.Errorf(
				"path entry %d cannot be empty",
				index,
			)
		}
	}

	return nil
}

func (c Config) EnvironmentNames() []string {
	names := make([]string, 0, len(c.Environment))

	for name := range c.Environment {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func (c Config) AliasNames() []string {
	names := make([]string, 0, len(c.Aliases))

	for name := range c.Aliases {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
