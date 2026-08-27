package config

import "testing"

func TestAdapterFor(t *testing.T) {
	tests := []string{
		"bash",
		"zsh",
		"fish",
		"nushell",
	}

	for _, shell := range tests {
		t.Run(shell, func(t *testing.T) {
			adapter, err := AdapterFor(shell)
			if err != nil {
				t.Fatalf("AdapterFor() returned error: %v", err)
			}

			if adapter.Name() != shell {
				t.Fatalf(
					"adapter.Name() = %q, want %q",
					adapter.Name(),
					shell,
				)
			}
		})
	}
}

func TestAdapterForRejectsUnsupportedShell(t *testing.T) {
	if _, err := AdapterFor("invalid"); err == nil {
		t.Fatal("AdapterFor() returned nil error")
	}
}
