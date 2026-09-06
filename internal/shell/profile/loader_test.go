package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIncludesSharedConfiguration(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(root, "shared"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(
		filepath.Join(root, "bash"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "shared", "config.toml"),
		[]byte(`
[environment]
EDITOR = "vim"

[path]
add = ["$HOME/bin"]

[aliases]
ll = "ls -lah"
`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	profile := []byte("mkcd() { cd \"$1\"; }\n")

	if err := os.WriteFile(
		filepath.Join(root, "bash", "server.bash"),
		profile,
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(root)

	got, err := loader.Load("bash", "server", true)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if got.Shared.Environment["EDITOR"] != "vim" {
		t.Fatalf(
			"EDITOR = %q, want %q",
			got.Shared.Environment["EDITOR"],
			"vim",
		)
	}

	if len(got.Shared.Path.Add) != 1 ||
		got.Shared.Path.Add[0] != "$HOME/bin" {
		t.Fatalf(
			"PATH = %#v, want %#v",
			got.Shared.Path.Add,
			[]string{"$HOME/bin"},
		)
	}

	if got.Shared.Aliases["ll"] != "ls -lah" {
		t.Fatalf(
			"alias ll = %q, want %q",
			got.Shared.Aliases["ll"],
			"ls -lah",
		)
	}

	if string(got.Profile) != string(profile) {
		t.Fatalf(
			"profile = %q, want %q",
			string(got.Profile),
			string(profile),
		)
	}
}

func TestLoadWithoutSharedConfiguration(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(root, "bash"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	profile := []byte("export EDITOR=vim\n")

	if err := os.WriteFile(
		filepath.Join(root, "bash", "server.bash"),
		profile,
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(root)

	got, err := loader.Load("bash", "server", false)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if got.Shared.Environment != nil ||
		got.Shared.Path.Add != nil ||
		got.Shared.Aliases != nil {
		t.Fatalf(
			"Shared = %#v, want empty configuration",
			got.Shared,
		)
	}

	if string(got.Profile) != string(profile) {
		t.Fatalf(
			"profile = %q, want %q",
			string(got.Profile),
			string(profile),
		)
	}
}

func TestLoadSharedConfigurationWithoutProfile(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(root, "shared"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "shared", "config.toml"),
		[]byte(`
[environment]
EDITOR = "vim"
`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(root)

	got, err := loader.Load("bash", "", true)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if got.Shared.Environment["EDITOR"] != "vim" {
		t.Fatalf(
			"EDITOR = %q, want %q",
			got.Shared.Environment["EDITOR"],
			"vim",
		)
	}

	if got.Profile != nil {
		t.Fatalf(
			"profile = %q, want nil",
			string(got.Profile),
		)
	}
}

func TestLoadMissingSharedConfigurationReturnsDescriptiveError(
	t *testing.T,
) {
	root := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(root, "bash"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "bash", "server.bash"),
		[]byte(""),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(root)

	_, err := loader.Load("bash", "server", true)
	if err == nil {
		t.Fatal("Load() returned nil error")
	}

	want := fmt.Sprintf(
		"shared shell configuration not found: %s",
		filepath.Join(root, "shared", "config.toml"),
	)

	if err.Error() != want {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			want,
		)
	}
}
