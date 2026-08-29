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
			name: "shell profile option",
			args: []string{
				"--shell-profile",
				"server",
			},
			wantOptions: cliOptions{
				ShellProfile: "server",
			},
		},
		{
			name: "shell profile equals option",
			args: []string{
				"--shell-profile=server",
			},
			wantOptions: cliOptions{
				ShellProfile: "server",
			},
		},
		{
			name: "shell profile option after command",
			args: []string{
				"shell",
				"--shell-profile",
				"server",
			},
			wantOptions: cliOptions{
				ShellProfile: "server",
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
		{
			name: "no shared rc option after command",
			args: []string{
				"shell",
				"--no-shared-rc",
			},
			wantOptions: cliOptions{
				NoSharedRC: true,
			},
			wantArgs: []string{
				"shell",
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

func TestParseCLIArgsRejectsMissingShellProfile(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--shell-profile",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}

	if err.Error() != "--shell-profile requires a profile name" {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			"--shell-profile requires a profile name",
		)
	}
}

func TestParseCLIArgsRejectsEmptyShellProfile(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--shell-profile=",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}

	if err.Error() != "--shell-profile requires a profile name" {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			"--shell-profile requires a profile name",
		)
	}
}

func TestParseCLIArgsRejectsWhitespaceShellProfile(t *testing.T) {
	_, _, err := parseCLIArgs([]string{
		"--shell-profile",
		"   ",
	})

	if err == nil {
		t.Fatal("parseCLIArgs() returned nil error")
	}

	if err.Error() != "--shell-profile requires a profile name" {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			"--shell-profile requires a profile name",
		)
	}
}

func TestParseCLIArgsExplicitShellProfileOverridesEnvironment(
	t *testing.T,
) {
	t.Setenv(
		shellProfileEnvName,
		"home",
	)

	options, _, err := parseCLIArgs([]string{
		"--shell-profile",
		"server",
	})
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.ShellProfile != "server" {
		t.Fatalf(
			"shell profile = %q, want %q",
			options.ShellProfile,
			"server",
		)
	}
}

func TestParseCLIArgsTrimsShellProfileEnvironment(t *testing.T) {
	t.Setenv(
		shellProfileEnvName,
		"  server  ",
	)

	options, _, err := parseCLIArgs(nil)
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.ShellProfile != "server" {
		t.Fatalf(
			"shell profile = %q, want %q",
			options.ShellProfile,
			"server",
		)
	}
}

func TestParseCLIArgsNoSharedRCDoesNotConsumeCommandArgument(
	t *testing.T,
) {
	options, commandArgs, err := parseCLIArgs([]string{
		"shell",
		"--no-shared-rc",
		"argument",
	})
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if !options.NoSharedRC {
		t.Fatal("NoSharedRC = false, want true")
	}

	wantArgs := []string{
		"shell",
		"argument",
	}

	if !reflect.DeepEqual(commandArgs, wantArgs) {
		t.Fatalf(
			"args = %#v, want %#v",
			commandArgs,
			wantArgs,
		)
	}
}

func TestParseCLIArgsUsesShellProfileEnvironment(t *testing.T) {
	t.Setenv(
		shellProfileEnvName,
		"server",
	)

	options, _, err := parseCLIArgs(nil)
	if err != nil {
		t.Fatalf(
			"parseCLIArgs() returned error: %v",
			err,
		)
	}

	if options.ShellProfile != "server" {
		t.Fatalf(
			"shell profile = %q, want %q",
			options.ShellProfile,
			"server",
		)
	}
}
