package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrUnavailable = errors.New(
	"multiplexer configuration is unavailable",
)

type Resolver struct {
	Stat func(string) (os.FileInfo, error)
	Home func() (string, error)
}

func NewResolver() *Resolver {
	return &Resolver{
		Stat: os.Stat,
		Home: os.UserHomeDir,
	}
}

func (r *Resolver) Tmux(runtimePath string) (string, error) {
	home, err := r.Home()
	if err != nil {
		return "", fmt.Errorf(
			"resolve home directory: %w",
			err,
		)
	}

	return r.resolve(
		filepath.Join(
			runtimePath,
			"config",
			"tmux",
			"tmux.conf",
		),
		filepath.Join(home, ".tmux.conf"),
		"/etc/tmux.conf",
	)
}

func (r *Resolver) Zellij(runtimePath string) (string, error) {
	home, err := r.Home()
	if err != nil {
		return "", fmt.Errorf(
			"resolve home directory: %w",
			err,
		)
	}

	return r.resolve(
		filepath.Join(
			runtimePath,
			"config",
			"zellij",
			"config.kdl",
		),
		filepath.Join(
			home,
			".config",
			"zellij",
			"config.kdl",
		),
		"/etc/zellij/config.kdl",
	)
}

func (r *Resolver) resolve(paths ...string) (string, error) {
	for _, path := range paths {
		info, err := r.Stat(path)

		if err == nil {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf(
					"%q: %w",
					path,
					ErrUnavailable,
				)
			}

			return path, nil
		}

		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf(
				"inspect multiplexer config %q: %w",
				path,
				err,
			)
		}
	}

	return "", nil
}
