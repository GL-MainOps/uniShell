package multiplexer

import "testing"

func TestResolveConfigUsesDefaults(t *testing.T) {
	t.Setenv(EnvMultiplexer, "")
	t.Setenv(EnvSession, "")
	t.Setenv(EnvMultiplexerSession, "")

	config := ResolveConfig("", "", "")

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
}

func TestResolveConfigUsesEnvironment(t *testing.T) {
	t.Setenv(EnvMultiplexer, "zellij")
	t.Setenv(EnvSession, "work")
	t.Setenv(EnvMultiplexerSession, "shell")

	config := ResolveConfig("", "", "")

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
}

func TestResolveConfigExplicitValuesOverrideEnvironment(t *testing.T) {
	t.Setenv(EnvMultiplexer, "zellij")
	t.Setenv(EnvSession, "environment")
	t.Setenv(EnvMultiplexerSession, "environment-session")

	config := ResolveConfig(
		"tmux",
		"explicit",
		"explicit-session",
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
}
