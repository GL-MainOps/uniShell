package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveUsesConfigurationBeforeSessionAndSystemEnvironment(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"CONFIG_VALUE":  "config",
			"SESSION_VALUE": "${SESSION_ONLY}",
			"SYSTEM_VALUE":  "${SYSTEM_ONLY}",
			"OVERRIDE":      "${CONFIG_VALUE}",
		},
	}

	session := map[string]string{
		"CONFIG_VALUE": "session",
		"SESSION_ONLY": "session",
		"OVERRIDE":     "session",
	}

	system := map[string]string{
		"CONFIG_VALUE": "system",
		"SYSTEM_ONLY":  "system",
		"OVERRIDE":     "system",
	}

	got, err := Resolve(cfg, session, system)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	want := Config{
		Environment: map[string]string{
			"CONFIG_VALUE":  "config",
			"SESSION_VALUE": "session",
			"SYSTEM_VALUE":  "system",
			"OVERRIDE":      "config",
		},
		Path:    PathConfig{},
		Aliases: map[string]string{},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Resolve() = %#v, want %#v", got, want)
	}
}

func TestResolveUsesSessionEnvironmentBeforeSystemEnvironment(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"VALUE": "${SHARED}",
		},
	}

	session := map[string]string{
		"SHARED": "session",
	}

	system := map[string]string{
		"SHARED": "system",
	}

	got, err := Resolve(cfg, session, system)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if want := "session"; got.Environment["VALUE"] != want {
		t.Fatalf(
			"resolved VALUE = %q, want %q",
			got.Environment["VALUE"],
			want,
		)
	}
}

func TestResolveFallsBackToSystemEnvironment(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"VALUE": "${HOME}",
		},
	}

	system := map[string]string{
		"HOME": "/home/test",
	}

	got, err := Resolve(cfg, nil, system)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if want := "/home/test"; got.Environment["VALUE"] != want {
		t.Fatalf(
			"resolved VALUE = %q, want %q",
			got.Environment["VALUE"],
			want,
		)
	}
}

func TestResolveSupportsRecursiveConfigurationReferences(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"BASE":   "${HOME}/.local",
			"BIN":    "${BASE}/bin",
			"CONFIG": "${BIN}/config",
		},
	}

	system := map[string]string{
		"HOME": "/home/test",
	}

	got, err := Resolve(cfg, nil, system)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	want := map[string]string{
		"BASE":   "/home/test/.local",
		"BIN":    "/home/test/.local/bin",
		"CONFIG": "/home/test/.local/bin/config",
	}

	if !reflect.DeepEqual(got.Environment, want) {
		t.Fatalf(
			"resolved environment = %#v, want %#v",
			got.Environment,
			want,
		)
	}
}

func TestResolveSupportsMultiplePlaceholders(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"VALUE": "${ONE}:${TWO}:${THREE}",
		},
	}

	session := map[string]string{
		"ONE":   "one",
		"TWO":   "two",
		"THREE": "three",
	}

	got, err := Resolve(cfg, session, nil)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if want := "one:two:three"; got.Environment["VALUE"] != want {
		t.Fatalf(
			"resolved VALUE = %q, want %q",
			got.Environment["VALUE"],
			want,
		)
	}
}

func TestResolveSupportsPathEntries(t *testing.T) {
	cfg := Config{
		Path: PathConfig{
			Add: []string{
				"${HOME}/.local/bin",
				"${UNISHELL_SESSION_RUNTIME_DIR}/bin",
			},
		},
	}

	session := map[string]string{
		"UNISHELL_SESSION_RUNTIME_DIR": "/runtime/session",
	}

	system := map[string]string{
		"HOME": "/home/test",
	}

	got, err := Resolve(cfg, session, system)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	want := []string{
		"/home/test/.local/bin",
		"/runtime/session/bin",
	}

	if !reflect.DeepEqual(got.Path.Add, want) {
		t.Fatalf(
			"resolved PATH entries = %#v, want %#v",
			got.Path.Add,
			want,
		)
	}
}

func TestResolveSupportsAliases(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"EDITOR": "nvim",
		},
		Aliases: map[string]string{
			"edit": "${EDITOR}",
			"root": "sudo ${EDITOR}",
		},
	}

	got, err := Resolve(cfg, nil, nil)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	want := map[string]string{
		"edit": "nvim",
		"root": "sudo nvim",
	}

	if !reflect.DeepEqual(got.Aliases, want) {
		t.Fatalf(
			"resolved aliases = %#v, want %#v",
			got.Aliases,
			want,
		)
	}
}

func TestResolveRejectsUnknownVariable(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"VALUE": "${DOES_NOT_EXIST}",
		},
	}

	_, err := Resolve(cfg, nil, nil)
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	want := `variable "DOES_NOT_EXIST" not found in configuration, session environment, or system environment`

	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %q, want substring %q",
			err.Error(),
			want,
		)
	}
}

func TestResolveRejectsInvalidVariableName(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"VALUE": "${BAD-NAME}",
		},
	}

	_, err := Resolve(cfg, nil, nil)
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	want := `invalid interpolation variable name "BAD-NAME"`

	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %q, want substring %q",
			err.Error(),
			want,
		)
	}
}

func TestResolveRejectsDirectCycle(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"A": "${A}",
		},
	}

	_, err := Resolve(cfg, nil, nil)
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	want := `cyclic reference involving "A"`

	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"error = %q, want substring %q",
			err.Error(),
			want,
		)
	}
}

func TestResolveRejectsIndirectCycle(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"A": "${B}",
			"B": "${C}",
			"C": "${A}",
		},
	}

	_, err := Resolve(cfg, nil, nil)
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	if !strings.Contains(err.Error(), "cyclic reference") {
		t.Fatalf(
			"error = %q, want cyclic reference error",
			err.Error(),
		)
	}
}

func TestResolveLeavesNonInterpolationSyntaxUntouched(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"VALUE": "$(whoami) $HOME",
		},
	}

	got, err := Resolve(cfg, nil, nil)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if want := "$(whoami) $HOME"; got.Environment["VALUE"] != want {
		t.Fatalf(
			"resolved VALUE = %q, want %q",
			got.Environment["VALUE"],
			want,
		)
	}
}

func TestResolveDoesNotMutateInput(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"BASE":  "${HOME}",
			"VALUE": "${BASE}/value",
		},
		Path: PathConfig{
			Add: []string{"${HOME}/bin"},
		},
		Aliases: map[string]string{
			"edit": "${BASE}/editor",
		},
	}

	original := Config{
		Environment: map[string]string{
			"BASE":  "${HOME}",
			"VALUE": "${BASE}/value",
		},
		Path: PathConfig{
			Add: []string{"${HOME}/bin"},
		},
		Aliases: map[string]string{
			"edit": "${BASE}/editor",
		},
	}

	_, err := Resolve(
		cfg,
		nil,
		map[string]string{
			"HOME": "/home/test",
		},
	)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if !reflect.DeepEqual(cfg, original) {
		t.Fatalf(
			"Resolve() mutated input config: got %#v, want %#v",
			cfg,
			original,
		)
	}
}
