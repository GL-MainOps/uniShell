package persistence

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gitlab.com/mainops/uniShell/internal/shell"
)

const ConfigFilename = ".unishell-config.toml"

type Config struct {
	Launch LaunchConfig `toml:"launch"`
}

type LaunchConfig struct {
	Configured             bool   `toml:"configured"`
	Shell                  string `toml:"shell"`
	ShellProfile           string `toml:"shell_profile"`
	NoSharedRC             bool   `toml:"no_shared_rc"`
	NoSharedRCSet          bool   `toml:"-"`
	Multiplexer            string `toml:"multiplexer"`
	SessionName            string `toml:"session_name"`
	MultiplexerSessionName string `toml:"multiplexer_session_name"`
	NewSession             bool   `toml:"new_session"`
}

func ConfigPath(root string) string {
	return filepath.Join(root, ConfigFilename)
}

func Read(root string) (Config, error) {
	data, err := os.ReadFile(ConfigPath(root))
	if err != nil {
		return Config{}, err
	}

	var config Config
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("invalid persistent configuration: %w; check setting names, sections, and value types", err)
	}

	var presence struct {
		Launch map[string]any `toml:"launch"`
	}
	if err := toml.Unmarshal(data, &presence); err != nil {
		return Config{}, fmt.Errorf("parse persistent configuration: %w", err)
	}
	_, config.Launch.NoSharedRCSet = presence.Launch["no_shared_rc"]
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if value := strings.ToLower(strings.TrimSpace(c.Launch.Shell)); value != "" {
		valid := false
		for _, name := range shell.SupportedShells() {
			if value == name {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid [launch].shell %q; choose bash, zsh, fish, or nushell", c.Launch.Shell)
		}
	}

	if value := strings.ToLower(strings.TrimSpace(c.Launch.Multiplexer)); value != "" {
		switch value {
		case "tmux", "zellij", "none", "disabled":
		default:
			return fmt.Errorf("invalid [launch].multiplexer %q; choose tmux, zellij, none, or disabled", c.Launch.Multiplexer)
		}
	}
	return nil
}

func Create(root string) (string, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", fmt.Errorf("create persistent runtime directory: %w", err)
	}
	if err := os.Chmod(root, 0700); err != nil {
		return "", fmt.Errorf("protect persistent runtime directory: %w", err)
	}

	path := ConfigPath(root)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect persistent configuration: %w", err)
	}

	if err := write(path, template(LaunchConfig{})); err != nil {
		return "", err
	}
	return path, nil
}

func SaveFirstLaunch(root string, launch LaunchConfig) error {
	config, err := Read(root)
	if err != nil {
		return fmt.Errorf("read persistent configuration: %w", err)
	}
	if config.Launch.Configured {
		return nil
	}
	launch.Configured = true
	path := ConfigPath(root)
	if err := write(path, template(launch)); err != nil {
		return err
	}
	return nil
}

func write(path string, data []byte) error {
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".unishell-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create persistent configuration: %w", err)
	}
	tempPath := file.Name()
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return fmt.Errorf("protect persistent configuration: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write persistent configuration: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync persistent configuration: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close persistent configuration: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish persistent configuration: %w", err)
	}
	return nil
}

func template(launch LaunchConfig) []byte {
	return []byte(fmt.Sprintf(`# uniShell persistent configuration.
#
# Edit values in [launch] to set persistent defaults. Command-line options
# and environment variables take precedence. Invalid keys and values are
# reported with the setting name when uniShell starts.
#
# Runtime directory: choose it during install with --runtime-dir or
# UNISHELL_RUNTIME_DIR. This file lives inside that directory.
# shell: bash, zsh, fish, or nushell.
# shell_profile: profile name from the selected shell's bundled profiles.
# no_shared_rc: true skips shared shell configuration; false loads it.
#   Override either value per launch with --no-shared-rc or --shared-rc.
# multiplexer: tmux, zellij, none, or disabled.
# session_name: optional uniShell session name.
# multiplexer_session_name: optional native multiplexer session name.
# new_session: true always starts a new multiplexer session.
#
# These are the corresponding commented examples:
# shell = "bash"
# shell_profile = "work"
# no_shared_rc = false
# multiplexer = "tmux"
# session_name = "work"
# multiplexer_session_name = "work"
# new_session = false
#
# Do not put UNISHELL_AUTH_TOKEN or other secrets in this file. uniShell
# stores its locally reusable authentication token in encrypted form.

[launch]
configured = %t
shell = %s
shell_profile = %s
no_shared_rc = %t # true skips shared shell configuration; false loads it.
multiplexer = %s # tmux, zellij, none, or disabled.
session_name = %s # Optional uniShell session name.
multiplexer_session_name = %s # Optional native multiplexer name.
new_session = %t # Always start a new multiplexer session when true.
`,
		launch.Configured,
		tomlString(launch.Shell),
		tomlString(launch.ShellProfile),
		launch.NoSharedRC,
		tomlString(launch.Multiplexer),
		tomlString(launch.SessionName),
		tomlString(launch.MultiplexerSessionName),
		launch.NewSession,
	))
}

func tomlString(value string) string {
	encoded, err := toml.Marshal(map[string]string{"value": value})
	if err != nil {
		return `""`
	}
	line, _, _ := strings.Cut(string(encoded), "\n")
	_, encodedValue, ok := strings.Cut(line, " = ")
	if !ok {
		return `""`
	}
	return encodedValue
}
