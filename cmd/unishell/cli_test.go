package main

import (
	"reflect"
	"testing"
)

func TestParseCLIArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantOptions cliOptions
		wantArgs    []string
	}{
		{
			name: "default shell",
			args: nil,
		},
		{
			name: "shell command",
			args: []string{
				"shell",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "global shell option",
			args: []string{
				"--shell",
				"zsh",
			},
			wantOptions: cliOptions{
				Shell: "zsh",
			},
		},
		{
			name: "shell command shell option",
			args: []string{
				"shell",
				"--shell",
				"zsh",
			},
			wantOptions: cliOptions{
				Shell: "zsh",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "global shell equals",
			args: []string{
				"--shell=zsh",
			},
			wantOptions: cliOptions{
				Shell: "zsh",
			},
		},
		{
			name: "shell command shell equals",
			args: []string{
				"shell",
				"--shell=zsh",
			},
			wantOptions: cliOptions{
				Shell: "zsh",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "runtime directory before command",
			args: []string{
				"--runtime-dir",
				"/tmp/unishell",
				"shell",
			},
			wantOptions: cliOptions{
				RuntimeDir: "/tmp/unishell",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "runtime directory after command",
			args: []string{
				"shell",
				"--runtime-dir",
				"/tmp/unishell",
			},
			wantOptions: cliOptions{
				RuntimeDir: "/tmp/unishell",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "both global options",
			args: []string{
				"--runtime-dir",
				"/tmp/unishell",
				"--shell",
				"fish",
				"shell",
			},
			wantOptions: cliOptions{
				RuntimeDir: "/tmp/unishell",
				Shell:      "fish",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "multiplexer option",
			args: []string{
				"--multiplexer",
				"zellij",
			},
			wantOptions: cliOptions{
				Multiplexer: "zellij",
			},
		},
		{
			name: "multiplexer equals option",
			args: []string{
				"--multiplexer=zellij",
			},
			wantOptions: cliOptions{
				Multiplexer: "zellij",
			},
		},
		{
			name: "multiplexer option after command",
			args: []string{
				"shell",
				"--multiplexer",
				"tmux",
			},
			wantOptions: cliOptions{
				Multiplexer: "tmux",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "command arguments remain untouched",
			args: []string{
				"version",
				"argument",
			},
			wantArgs: []string{
				"version",
				"argument",
			},
		},
		{
			name: "shell profile option",
			args: []string{
				"--shell-profile",
				"work",
			},
			wantOptions: cliOptions{
				ShellProfile: "work",
			},
		},
		{
			name: "shell profile equals option",
			args: []string{
				"--shell-profile=work",
			},
			wantOptions: cliOptions{
				ShellProfile: "work",
			},
		},
		{
			name: "shell profile option after command",
			args: []string{
				"shell",
				"--shell-profile",
				"work",
			},
			wantOptions: cliOptions{
				ShellProfile: "work",
			},
			wantArgs: []string{
				"shell",
			},
		},
		{
			name: "no shared rc option",
			args: []string{
				"--no-shared-rc",
			},
			wantOptions: cliOptions{
				NoSharedRC: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotOptions, gotArgs, err := parseCLIArgs(test.args)
			if err != nil {
				t.Fatalf(
					"parseCLIArgs() returned error: %v",
					err,
				)
			}

			if !reflect.DeepEqual(
				gotOptions,
				test.wantOptions,
			) {
				t.Fatalf(
					"options = %#v, want %#v",
					gotOptions,
					test.wantOptions,
				)
			}

			if !reflect.DeepEqual(
				gotArgs,
				test.wantArgs,
			) {
				t.Fatalf(
					"args = %#v, want %#v",
					gotArgs,
					test.wantArgs,
				)
			}
		})
	}
}

func TestParseCLIArgsRejectsMissingShell(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--shell",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsEmptyShell(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--shell=",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsMissingRuntimeDirectory(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--runtime-dir",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsEmptyRuntimeDirectory(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--runtime-dir=",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsUsesMultiplexerEnvironment(t *testing.T) {
	t.Setenv(
		multiplexerEnvName,
		"zellij",
	)

	options, _, err := parseCLIArgs(nil)
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.Multiplexer != "zellij" {
		t.Fatalf(
			"multiplexer = %q, want %q",
			options.Multiplexer,
			"zellij",
		)
	}
}

func TestParseCLIArgsExplicitMultiplexerOverridesEnvironment(
	t *testing.T,
) {
	t.Setenv(
		multiplexerEnvName,
		"tmux",
	)

	options, _, err := parseCLIArgs([]string{
		"--multiplexer",
		"zellij",
	})
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.Multiplexer != "zellij" {
		t.Fatalf(
			"multiplexer = %q, want %q",
			options.Multiplexer,
			"zellij",
		)
	}
}

func TestParseCLIArgsRejectsMissingMultiplexer(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--multiplexer",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsEmptyMultiplexer(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--multiplexer=",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsUsesSessionEnvironment(t *testing.T) {
	t.Setenv(sessionEnvName, "work")

	options, _, err := parseCLIArgs(nil)
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.SessionName != "work" {
		t.Fatalf(
			"session name = %q, want %q",
			options.SessionName,
			"work",
		)
	}
}

func TestParseCLIArgsUsesMultiplexerSessionEnvironment(t *testing.T) {
	t.Setenv(
		multiplexerSessionEnvName,
		"dev",
	)

	options, _, err := parseCLIArgs(nil)
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.MultiplexerSessionName != "dev" {
		t.Fatalf(
			"multiplexer session name = %q, want %q",
			options.MultiplexerSessionName,
			"dev",
		)
	}
}

func TestParseCLIArgsExplicitSessionOverridesEnvironment(t *testing.T) {
	t.Setenv(sessionEnvName, "environment")

	options, _, err := parseCLIArgs([]string{
		"--session",
		"cli",
	})
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.SessionName != "cli" {
		t.Fatalf(
			"session name = %q, want %q",
			options.SessionName,
			"cli",
		)
	}
}

func TestParseCLIArgsExplicitMultiplexerSessionOverridesEnvironment(
	t *testing.T,
) {
	t.Setenv(
		multiplexerSessionEnvName,
		"environment",
	)

	options, _, err := parseCLIArgs([]string{
		"--multiplexer-session",
		"cli",
	})
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.MultiplexerSessionName != "cli" {
		t.Fatalf(
			"multiplexer session name = %q, want %q",
			options.MultiplexerSessionName,
			"cli",
		)
	}
}

func TestParseCLIArgsRejectsMissingSession(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--session",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsEmptySession(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--session=",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsMissingMultiplexerSession(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--multiplexer-session",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsEmptyMultiplexerSession(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--multiplexer-session=",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsUsesShellProfileEnvironment(t *testing.T) {
	t.Setenv(
		shellProfileEnvName,
		"environment",
	)

	options, _, err := parseCLIArgs(nil)
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.ShellProfile != "environment" {
		t.Fatalf(
			"shell profile = %q, want %q",
			options.ShellProfile,
			"environment",
		)
	}
}

func TestParseCLIArgsExplicitShellProfileOverridesEnvironment(
	t *testing.T,
) {
	t.Setenv(
		shellProfileEnvName,
		"environment",
	)

	options, _, err := parseCLIArgs([]string{
		"--shell-profile",
		"cli",
	})
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.ShellProfile != "cli" {
		t.Fatalf(
			"shell profile = %q, want %q",
			options.ShellProfile,
			"cli",
		)
	}
}

func TestParseCLIArgsRejectsMissingShellProfile(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--shell-profile",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}

func TestParseCLIArgsRejectsEmptyShellProfile(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--shell-profile=",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}
}
