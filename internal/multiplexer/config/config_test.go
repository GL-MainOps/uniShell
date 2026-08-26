package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolverPrefersBundledTmuxConfig(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	bundled := filepath.Join(
		root,
		"config",
		"tmux",
		"tmux.conf",
	)

	if err := os.MkdirAll(
		filepath.Dir(bundled),
		0700,
	); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	if err := os.WriteFile(
		bundled,
		[]byte("set -g mouse on\n"),
		0600,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	userConfig := filepath.Join(home, ".tmux.conf")

	if err := os.WriteFile(
		userConfig,
		[]byte("set -g mouse off\n"),
		0600,
	); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	resolver := NewResolver()
	resolver.Home = func() (string, error) {
		return home, nil
	}

	got, err := resolver.Tmux(root)
	if err != nil {
		t.Fatalf("Tmux() returned error: %v", err)
	}

	if got != bundled {
		t.Fatalf(
			"config = %q, want %q",
			got,
			bundled,
		)
	}
}

func TestResolverFallsBackToUserTmuxConfig(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	userConfig := filepath.Join(home, ".tmux.conf")

	if err := os.WriteFile(
		userConfig,
		[]byte("set -g mouse on\n"),
		0600,
	); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	resolver := NewResolver()
	resolver.Home = func() (string, error) {
		return home, nil
	}

	got, err := resolver.Tmux(root)
	if err != nil {
		t.Fatalf("Tmux() returned error: %v", err)
	}

	if got != userConfig {
		t.Fatalf(
			"config = %q, want %q",
			got,
			userConfig,
		)
	}
}

func TestResolverFallsBackToSystemTmuxConfig(t *testing.T) {
	resolver := NewResolver()

	systemConfig := filepath.Join(
		t.TempDir(),
		"tmux.conf",
	)

	if err := os.WriteFile(
		systemConfig,
		[]byte("set -g mouse on\n"),
		0600,
	); err != nil {
		t.Fatalf("write system config: %v", err)
	}

	got, err := resolver.resolve(
		filepath.Join(t.TempDir(), "bundled"),
		filepath.Join(t.TempDir(), "user"),
		systemConfig,
	)
	if err != nil {
		t.Fatalf("resolve() returned error: %v", err)
	}

	if got != systemConfig {
		t.Fatalf(
			"config = %q, want %q",
			got,
			systemConfig,
		)
	}
}

func TestResolverReturnsEmptyWhenNoConfigExists(t *testing.T) {
	resolver := NewResolver()

	got, err := resolver.resolve(
		filepath.Join(t.TempDir(), "bundled"),
		filepath.Join(t.TempDir(), "user"),
		filepath.Join(t.TempDir(), "system"),
	)
	if err != nil {
		t.Fatalf("resolve() returned error: %v", err)
	}

	if got != "" {
		t.Fatalf(
			"config = %q, want empty",
			got,
		)
	}
}

func TestResolverRejectsNonRegularConfig(t *testing.T) {
	directory := filepath.Join(
		t.TempDir(),
		"tmux.conf",
	)

	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatalf("create directory: %v", err)
	}

	resolver := NewResolver()

	_, err := resolver.resolve(directory)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf(
			"resolve() error = %v, want %v",
			err,
			ErrUnavailable,
		)
	}
}

func TestResolverPrefersBundledZellijConfig(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	bundled := filepath.Join(
		root,
		"config",
		"zellij",
		"config.kdl",
	)

	if err := os.MkdirAll(
		filepath.Dir(bundled),
		0700,
	); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	if err := os.WriteFile(
		bundled,
		[]byte("pane_frames false\n"),
		0600,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	userConfig := filepath.Join(
		home,
		".config",
		"zellij",
		"config.kdl",
	)

	if err := os.MkdirAll(
		filepath.Dir(userConfig),
		0700,
	); err != nil {
		t.Fatalf("create user config directory: %v", err)
	}

	if err := os.WriteFile(
		userConfig,
		[]byte("pane_frames true\n"),
		0600,
	); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	resolver := NewResolver()
	resolver.Home = func() (string, error) {
		return home, nil
	}

	got, err := resolver.Zellij(root)
	if err != nil {
		t.Fatalf("Zellij() returned error: %v", err)
	}

	if got != bundled {
		t.Fatalf(
			"config = %q, want %q",
			got,
			bundled,
		)
	}
}
