package multiplexer

import "testing"

func TestDefaultRegistryContainsSupportedMultiplexers(t *testing.T) {
	registry := DefaultRegistry()

	for _, name := range []string{"tmux", "zellij"} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf(
				"default registry does not contain %q",
				name,
			)
		}
	}
}
