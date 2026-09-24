package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
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
	if err := toml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse persistent configuration: %w", err)
	}
	return config, nil
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
# The options below are examples and documentation. Uncomment or edit the
# active values in [launch] to set defaults. Command-line options and
# environment variables override these defaults.
#
# Available options:
# runtime_dir = "PATH"                 # Choose at install with --runtime-dir or UNISHELL_RUNTIME_DIR.
# shell = "bash"                       # bash, zsh, fish, or nushell.
# shell_profile = "work"              # Select a bundled shell profile.
# no_shared_rc = false                 # Skip shared shell configuration.
# multiplexer = "tmux"                 # tmux, zellij, none, or disabled.
# session_name = "work"                # Optional uniShell session name.
# multiplexer_session_name = "work"   # Optional native multiplexer name.
# new_session = false                  # Always start a new multiplexer session.
#
# Do not put UNISHELL_AUTH_TOKEN or other secrets in this file. uniShell
# stores its locally reusable authentication token in encrypted form.

[launch]
configured = %t
shell = %s
shell_profile = %s
no_shared_rc = %t
multiplexer = %s
session_name = %s
multiplexer_session_name = %s
new_session = %t
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
