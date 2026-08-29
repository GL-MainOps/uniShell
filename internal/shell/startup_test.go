package shell

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildStartupBash(t *testing.T) {
	startup, err := BuildStartup(
		"bash",
		"/runtime/config/shell/server.bash",
		"/runtime",
	)
	if err != nil {
		t.Fatalf(
			"BuildStartup() returned error: %v",
			err,
		)
	}

	want := Startup{
		Args: []string{
			"--noprofile",
			"--rcfile",
			"/runtime/config/shell/server.bash",
		},
	}

	if !reflect.DeepEqual(startup, want) {
		t.Fatalf(
			"startup = %#v, want %#v",
			startup,
			want,
		)
	}
}

func TestBuildStartupZsh(t *testing.T) {
	runtimeDir := t.TempDir()

	startup, err := BuildStartup(
		"zsh",
		"/runtime/config/shell/server.zsh",
		runtimeDir,
	)
	if err != nil {
		t.Fatalf(
			"BuildStartup() returned error: %v",
			err,
		)
	}

	wantZDOTDIR := filepath.Join(
		runtimeDir,
		"config",
		"shell",
		"zsh",
	)

	if startup.Env["ZDOTDIR"] != wantZDOTDIR {
		t.Fatalf(
			"ZDOTDIR = %q, want %q",
			startup.Env["ZDOTDIR"],
			wantZDOTDIR,
		)
	}

	wantArgs := []string{"-d"}

	if !reflect.DeepEqual(startup.Args, wantArgs) {
		t.Fatalf(
			"args = %#v, want %#v",
			startup.Args,
			wantArgs,
		)
	}

	zshrc := filepath.Join(
		wantZDOTDIR,
		".zshrc",
	)

	data, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf(
			"ReadFile(%q) returned error: %v",
			zshrc,
			err,
		)
	}

	wantContent := "source '/runtime/config/shell/server.zsh'\n"

	if string(data) != wantContent {
		t.Fatalf(
			"zshrc = %q, want %q",
			string(data),
			wantContent,
		)
	}
}

func TestBuildStartupFish(t *testing.T) {
	startup, err := BuildStartup(
		"fish",
		"/runtime/config/shell/server.fish",
		"/runtime",
	)
	if err != nil {
		t.Fatalf(
			"BuildStartup() returned error: %v",
			err,
		)
	}

	want := Startup{
		Args: []string{
			"--no-config",
			"--init-command",
			"source '/runtime/config/shell/server.fish'",
		},
	}

	if !reflect.DeepEqual(startup, want) {
		t.Fatalf(
			"startup = %#v, want %#v",
			startup,
			want,
		)
	}
}

func TestBuildStartupNushell(t *testing.T) {
	startup, err := BuildStartup(
		"nushell",
		"/runtime/config/shell/server.nushell",
		"/runtime",
	)
	if err != nil {
		t.Fatalf(
			"BuildStartup() returned error: %v",
			err,
		)
	}

	want := Startup{
		Args: []string{
			"--config",
			"/runtime/config/shell/server.nushell",
		},
	}

	if !reflect.DeepEqual(startup, want) {
		t.Fatalf(
			"startup = %#v, want %#v",
			startup,
			want,
		)
	}
}

func TestBuildStartupRejectsUnsupportedShell(t *testing.T) {
	_, err := BuildStartup(
		"invalid",
		"/runtime/config/shell/server.invalid",
		"/runtime",
	)

	if err == nil {
		t.Fatal("BuildStartup() returned nil error")
	}

	want := `unsupported shell: "invalid"`

	if err.Error() != want {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			want,
		)
	}
}

func TestBuildStartupRejectsEmptyInputs(t *testing.T) {
	tests := []struct {
		name        string
		shellName   string
		startupPath string
		runtimeDir  string
		want        string
	}{
		{
			name:        "empty shell",
			startupPath: "/runtime/config/shell/server.bash",
			runtimeDir:  "/runtime",
			want:        "shell name cannot be empty",
		},
		{
			name:       "empty startup path",
			shellName:  "bash",
			runtimeDir: "/runtime",
			want:       "shell startup path cannot be empty",
		},
		{
			name:        "empty runtime directory",
			shellName:   "bash",
			startupPath: "/runtime/config/shell/server.bash",
			want:        "shell startup runtime directory cannot be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildStartup(
				test.shellName,
				test.startupPath,
				test.runtimeDir,
			)

			if err == nil {
				t.Fatal("BuildStartup() returned nil error")
			}

			if err.Error() != test.want {
				t.Fatalf(
					"error = %q, want %q",
					err.Error(),
					test.want,
				)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	value := "/runtime/it's/config"

	got := shellQuote(value)
	want := "'/runtime/it'\"'\"'s/config'"

	if got != want {
		t.Fatalf(
			"shellQuote(%q) = %q, want %q",
			value,
			got,
			want,
		)
	}
}

func TestBuildStartupZshDoesNotFollowExistingHostZDOTDIR(
	t *testing.T,
) {
	t.Setenv("ZDOTDIR", "/host/zsh")

	runtimeDir := t.TempDir()

	startup, err := BuildStartup(
		"zsh",
		"/runtime/profile.zsh",
		runtimeDir,
	)
	if err != nil {
		t.Fatalf(
			"BuildStartup() returned error: %v",
			err,
		)
	}

	if strings.Contains(
		startup.Env["ZDOTDIR"],
		"/host/zsh",
	) {
		t.Fatalf(
			"ZDOTDIR unexpectedly inherited host value: %q",
			startup.Env["ZDOTDIR"],
		)
	}
}
