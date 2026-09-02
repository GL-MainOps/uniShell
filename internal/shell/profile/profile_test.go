package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProfile(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(root, "bash"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	profilePath := filepath.Join(
		root,
		"bash",
		"server.bash",
	)

	if err := os.WriteFile(
		profilePath,
		[]byte("# bash profile\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	resolver := NewResolver(root)

	got, err := resolver.Resolve("bash", "server")
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	wantProfile := profilePath
	if got.ProfilePath != wantProfile {
		t.Fatalf(
			"ProfilePath = %q, want %q",
			got.ProfilePath,
			wantProfile,
		)
	}

	wantShared := filepath.Join(
		root,
		"shared",
		"config.toml",
	)

	if got.SharedPath != wantShared {
		t.Fatalf(
			"SharedPath = %q, want %q",
			got.SharedPath,
			wantShared,
		)
	}
}

func TestResolveProfileUsesShellSpecificExtensions(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		shell     string
		extension string
	}{
		{
			shell:     "bash",
			extension: "bash",
		},
		{
			shell:     "zsh",
			extension: "zsh",
		},
		{
			shell:     "fish",
			extension: "fish",
		},
		{
			shell:     "nushell",
			extension: "nu",
		},
	}

	resolver := NewResolver(root)

	for _, test := range tests {
		t.Run(test.shell, func(t *testing.T) {
			profileDir := filepath.Join(root, test.shell)

			if err := os.MkdirAll(profileDir, 0o755); err != nil {
				t.Fatal(err)
			}

			profilePath := filepath.Join(
				profileDir,
				"server."+test.extension,
			)

			if err := os.WriteFile(
				profilePath,
				[]byte("# shell profile\n"),
				0o644,
			); err != nil {
				t.Fatal(err)
			}

			got, err := resolver.Resolve(test.shell, "server")
			if err != nil {
				t.Fatalf(
					"Resolve() returned error: %v",
					err,
				)
			}

			if got.ProfilePath != profilePath {
				t.Fatalf(
					"ProfilePath = %q, want %q",
					got.ProfilePath,
					profilePath,
				)
			}
		})
	}
}

func TestResolveRejectsUnsupportedShell(t *testing.T) {
	root := t.TempDir()

	resolver := NewResolver(root)

	_, err := resolver.Resolve("unsupported", "server")
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	want := `unsupported shell: "unsupported"`

	if err.Error() != want {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			want,
		)
	}
}

func TestResolveWithoutProfile(t *testing.T) {
	resolver := NewResolver(t.TempDir())

	got, err := resolver.Resolve("bash", "")
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got != (Result{}) {
		t.Fatalf(
			"Resolve() = %#v, want empty result",
			got,
		)
	}
}

func TestResolveMissingProfileReturnsDescriptiveError(t *testing.T) {
	resolver := NewResolver(t.TempDir())

	_, err := resolver.Resolve("bash", "server")
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	want := `shell profile "server" not found for shell "bash"`

	if err.Error() != want {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			want,
		)
	}
}

func TestResolveRejectsNonRegularProfile(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(root, "bash", "server.bash"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	resolver := NewResolver(root)

	_, err := resolver.Resolve("bash", "server")
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	want := `shell profile "server" for shell "bash" is not a regular file`

	if err.Error() != want {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			want,
		)
	}
}

func TestResolveRejectsEmptyShell(t *testing.T) {
	resolver := NewResolver(t.TempDir())

	_, err := resolver.Resolve("", "server")
	if err == nil {
		t.Fatal("Resolve() returned nil error")
	}

	if err.Error() != "shell cannot be empty" {
		t.Fatalf("error = %q", err.Error())
	}
}
