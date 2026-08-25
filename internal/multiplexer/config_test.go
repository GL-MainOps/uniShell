package multiplexer

import "testing"

func TestResolveConfigUsesDefaults(t *testing.T) {
	t.Setenv(EnvMultiplexer, "")
	t.Setenv(EnvSession, "")
	t.Setenv(EnvMultiplexerSession, "")
	t.Setenv(EnvTmuxSocket, "")

	config := ResolveConfig("", "", "", "")

	if config.Name != "tmux" {
		t.Fatalf("multiplexer = %q, want %q", config.Name, "tmux")
	}

	if config.Session != "default" {
		t.Fatalf("session = %q, want %q", config.Session, "default")
	}

	if config.MultiplexerName != "" {
		t.Fatalf(
			"multiplexer session = %q, want empty",
			config.MultiplexerName,
		)
	}

	if config.TmuxSocket != "" {
		t.Fatalf(
			"tmux socket = %q, want empty",
			config.TmuxSocket,
		)
	}
}

func TestResolveConfigUsesEnvironment(t *testing.T) {
	t.Setenv(EnvMultiplexer, "zellij")
	t.Setenv(EnvSession, "work")
	t.Setenv(EnvMultiplexerSession, "shell")
	t.Setenv(EnvTmuxSocket, "/tmp/custom-tmux.sock")

	config := ResolveConfig("", "", "", "")

	if config.Name != "zellij" {
		t.Fatalf("multiplexer = %q, want %q", config.Name, "zellij")
	}

	if config.Session != "work" {
		t.Fatalf("session = %q, want %q", config.Session, "work")
	}

	if config.MultiplexerName != "shell" {
		t.Fatalf(
			"multiplexer session = %q, want %q",
			config.MultiplexerName,
			"shell",
		)
	}

	if config.TmuxSocket != "/tmp/custom-tmux.sock" {
		t.Fatalf(
			"tmux socket = %q, want %q",
			config.TmuxSocket,
			"/tmp/custom-tmux.sock",
		)
	}
}

func TestResolveConfigExplicitValuesOverrideEnvironment(t *testing.T) {
	t.Setenv(EnvMultiplexer, "zellij")
	t.Setenv(EnvSession, "environment")
	t.Setenv(EnvMultiplexerSession, "environment-session")
	t.Setenv(EnvTmuxSocket, "/tmp/environment.sock")

	config := ResolveConfig(
		"tmux",
		"explicit",
		"explicit-session",
		"/tmp/explicit.sock",
	)

	if config.Name != "tmux" {
		t.Fatalf("multiplexer = %q, want %q", config.Name, "tmux")
	}

	if config.Session != "explicit" {
		t.Fatalf("session = %q, want %q", config.Session, "explicit")
	}

	if config.MultiplexerName != "explicit-session" {
		t.Fatalf(
			"multiplexer session = %q, want %q",
			config.MultiplexerName,
			"explicit-session",
		)
	}

	if config.TmuxSocket != "/tmp/explicit.sock" {
		t.Fatalf(
			"tmux socket = %q, want %q",
			config.TmuxSocket,
			"/tmp/explicit.sock",
		)
	}
}
