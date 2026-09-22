package config

import (
	"fmt"
	"regexp"
)

var interpolationPattern = regexp.MustCompile(`\$\{([^}]*)\}`)

type interpolationResolver struct {
	config    map[string]string
	session   map[string]string
	system    map[string]string
	resolving map[string]bool
}

func Resolve(
	cfg Config,
	sessionEnvironment map[string]string,
	systemEnvironment map[string]string,
) (Config, error) {
	resolver := interpolationResolver{
		config:    cfg.Environment,
		session:   sessionEnvironment,
		system:    systemEnvironment,
		resolving: make(map[string]bool),
	}

	resolved := Config{
		Environment: make(map[string]string, len(cfg.Environment)),
		Path:        PathConfig{},
		Aliases:     make(map[string]string, len(cfg.Aliases)),
	}

	for _, name := range cfg.EnvironmentNames() {
		value, err := resolver.resolveConfigVariable(name)
		if err != nil {
			return Config{}, fmt.Errorf(
				"resolve environment variable %q: %w",
				name,
				err,
			)
		}

		resolved.Environment[name] = value
	}

	if cfg.Path.Add != nil {
		resolved.Path.Add = make([]string, len(cfg.Path.Add))
	}

	for index, value := range cfg.Path.Add {
		resolvedValue, err := resolver.resolveValue(value)
		if err != nil {
			return Config{}, fmt.Errorf(
				"resolve path entry %d: %w",
				index,
				err,
			)
		}

		resolved.Path.Add[index] = resolvedValue
	}

	for _, name := range cfg.AliasNames() {
		value, err := resolver.resolveValue(cfg.Aliases[name])
		if err != nil {
			return Config{}, fmt.Errorf(
				"resolve alias %q: %w",
				name,
				err,
			)
		}

		resolved.Aliases[name] = value
	}

	return resolved, nil
}

func (r *interpolationResolver) resolveConfigVariable(
	name string,
) (string, error) {
	if r.resolving[name] {
		return "", fmt.Errorf(
			"cyclic reference involving %q",
			name,
		)
	}

	value, ok := r.config[name]
	if !ok {
		return "", fmt.Errorf(
			"configuration variable %q not found",
			name,
		)
	}

	r.resolving[name] = true
	defer delete(r.resolving, name)

	return r.resolveValue(value)
}

func (r *interpolationResolver) resolveValue(
	value string,
) (string, error) {
	var firstErr error

	resolved := interpolationPattern.ReplaceAllStringFunc(
		value,
		func(match string) string {
			if firstErr != nil {
				return match
			}

			submatches := interpolationPattern.FindStringSubmatch(match)

			if len(submatches) != 2 {
				firstErr = fmt.Errorf(
					"invalid interpolation %q",
					match,
				)
				return match
			}

			name := submatches[1]

			if !validIdentifier(name) {
				firstErr = fmt.Errorf(
					"invalid interpolation variable name %q",
					name,
				)
				return match
			}

			resolvedValue, err := r.resolveVariable(name)
			if err != nil {
				firstErr = err
				return match
			}

			return resolvedValue
		},
	)

	if firstErr != nil {
		return "", firstErr
	}

	return resolved, nil
}

func (r *interpolationResolver) resolveVariable(
	name string,
) (string, error) {
	if _, ok := r.config[name]; ok {
		return r.resolveConfigVariable(name)
	}

	if value, ok := r.session[name]; ok {
		return value, nil
	}

	if value, ok := r.system[name]; ok {
		return value, nil
	}

	return "", fmt.Errorf(
		"variable %q not found in configuration, session environment, or system environment",
		name,
	)
}
