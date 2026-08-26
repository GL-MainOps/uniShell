package multiplexer

import (
	"reflect"
	"strings"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestParseOptionsFromEnvironmentUsesTmuxOptions(
	t *testing.T,
) {
	t.Setenv(
		tmuxOptionsEnvironmentVariable,
		`-f "/tmp/my config" -L work`,
	)
	t.Setenv(
		zellijOptionsEnvironmentVariable,
		"",
	)

	got, err := ParseOptionsFromEnvironment()
	if err != nil {
		t.Fatalf(
			"ParseOptionsFromEnvironment() returned error: %v",
			err,
		)
	}

	want := api.Options{
		Tmux: api.TmuxOptions{
			CreateArgs: []string{
				"-f",
				"/tmp/my config",
				"-L",
				"work",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"options = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestParseOptionsFromEnvironmentUsesZellijOptions(
	t *testing.T,
) {
	t.Setenv(
		tmuxOptionsEnvironmentVariable,
		"",
	)
	t.Setenv(
		zellijOptionsEnvironmentVariable,
		`--layout "compact layout.kdl"`,
	)

	got, err := ParseOptionsFromEnvironment()
	if err != nil {
		t.Fatalf(
			"ParseOptionsFromEnvironment() returned error: %v",
			err,
		)
	}

	want := api.Options{
		Zellij: api.ZellijOptions{
			CreateArgs: []string{
				"--layout",
				"compact layout.kdl",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"options = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestParseOptionsFromEnvironmentUsesBothBackends(
	t *testing.T,
) {
	t.Setenv(
		tmuxOptionsEnvironmentVariable,
		`-L work`,
	)
	t.Setenv(
		zellijOptionsEnvironmentVariable,
		`--layout compact.kdl`,
	)

	got, err := ParseOptionsFromEnvironment()
	if err != nil {
		t.Fatalf(
			"ParseOptionsFromEnvironment() returned error: %v",
			err,
		)
	}

	want := api.Options{
		Tmux: api.TmuxOptions{
			CreateArgs: []string{
				"-L",
				"work",
			},
		},
		Zellij: api.ZellijOptions{
			CreateArgs: []string{
				"--layout",
				"compact.kdl",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"options = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestParseOptionsFromEnvironmentAllowsUnsetOptions(
	t *testing.T,
) {
	t.Setenv(
		tmuxOptionsEnvironmentVariable,
		"",
	)
	t.Setenv(
		zellijOptionsEnvironmentVariable,
		"",
	)

	got, err := ParseOptionsFromEnvironment()
	if err != nil {
		t.Fatalf(
			"ParseOptionsFromEnvironment() returned error: %v",
			err,
		)
	}

	if !reflect.DeepEqual(got, api.Options{}) {
		t.Fatalf(
			"options = %#v, want zero-value options",
			got,
		)
	}
}

func TestParseOptionsFromEnvironmentRejectsMalformedTmuxOptions(
	t *testing.T,
) {
	t.Setenv(
		tmuxOptionsEnvironmentVariable,
		`--name "unterminated`,
	)
	t.Setenv(
		zellijOptionsEnvironmentVariable,
		"",
	)

	_, err := ParseOptionsFromEnvironment()
	if err == nil {
		t.Fatal(
			"ParseOptionsFromEnvironment() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		tmuxOptionsEnvironmentVariable,
	) {
		t.Fatalf(
			"error = %q, want environment variable name",
			err,
		)
	}
}

func TestParseOptionsFromEnvironmentRejectsMalformedZellijOptions(
	t *testing.T,
) {
	t.Setenv(
		tmuxOptionsEnvironmentVariable,
		"",
	)
	t.Setenv(
		zellijOptionsEnvironmentVariable,
		`--layout "unterminated`,
	)

	_, err := ParseOptionsFromEnvironment()
	if err == nil {
		t.Fatal(
			"ParseOptionsFromEnvironment() returned nil error",
		)
	}

	if !strings.Contains(
		err.Error(),
		zellijOptionsEnvironmentVariable,
	) {
		t.Fatalf(
			"error = %q, want environment variable name",
			err,
		)
	}
}
