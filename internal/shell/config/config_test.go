package config

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	input := []byte(`
[environment]
EDITOR = "vim"
VISUAL = "$EDITOR"

[path]
add = [
    "$HOME/.local/bin",
    "$HOME/bin",
]

[aliases]
ll = "ls -lah"
gs = "git status"
`)

	got, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	want := Config{
		Environment: map[string]string{
			"EDITOR": "vim",
			"VISUAL": "$EDITOR",
		},
		Path: PathConfig{
			Add: []string{
				"$HOME/.local/bin",
				"$HOME/bin",
			},
		},
		Aliases: map[string]string{
			"ll": "ls -lah",
			"gs": "git status",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse() = %#v, want %#v", got, want)
	}
}

func TestParseRejectsEmptyPathEntry(t *testing.T) {
	_, err := Parse([]byte(`
[path]
add = [""]
`))
	if err == nil {
		t.Fatal("Parse() returned nil error")
	}
}

func TestNamesAreSorted(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"Z": "z",
			"A": "a",
		},
		Aliases: map[string]string{
			"z": "z",
			"a": "a",
		},
	}

	if got, want := cfg.EnvironmentNames(), []string{"A", "Z"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EnvironmentNames() = %#v, want %#v", got, want)
	}

	if got, want := cfg.AliasNames(), []string{"a", "z"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("AliasNames() = %#v, want %#v", got, want)
	}
}

func TestValidateRejectsInvalidEnvironmentName(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"BAD-NAME": "value",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() returned nil error")
	}
}

func TestValidateRejectsInvalidAliasName(t *testing.T) {
	cfg := Config{
		Aliases: map[string]string{
			"bad alias": "value",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() returned nil error")
	}
}

func TestValidateAcceptsValidIdentifiers(t *testing.T) {
	cfg := Config{
		Environment: map[string]string{
			"EDITOR":       "vim",
			"_PRIVATE":     "value",
			"FZF_DEFAULTS": "value",
		},
		Aliases: map[string]string{
			"ll":     "ls -lah",
			"_debug": "true",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
}
