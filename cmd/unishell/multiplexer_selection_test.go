package main

import "testing"

func TestParseMultiplexerSelection(t *testing.T) {
	tests := []struct {
		input     string
		want      string
		valid     bool
		cancelled bool
	}{
		{
			input: "1",
			want:  multiplexerTmux,
			valid: true,
		},
		{
			input: "tmux",
			want:  multiplexerTmux,
			valid: true,
		},
		{
			input: "2",
			want:  multiplexerZellij,
			valid: true,
		},
		{
			input: "zellij",
			want:  multiplexerZellij,
			valid: true,
		},
		{
			input: "3",
			want:  multiplexerNone,
			valid: true,
		},
		{
			input: "4",
		},
		{
			input: "none",
			want:  multiplexerNone,
			valid: true,
		},
		{
			input: "disabled",
			want:  multiplexerNone,
			valid: true,
		},
		{
			input:     "q",
			cancelled: true,
		},
		{
			input:     "quit",
			cancelled: true,
		},
		{
			input: "invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, valid, cancelled :=
				parseMultiplexerSelection(test.input)

			if got != test.want {
				t.Fatalf(
					"selected = %q, want %q",
					got,
					test.want,
				)
			}

			if valid != test.valid {
				t.Fatalf(
					"valid = %v, want %v",
					valid,
					test.valid,
				)
			}

			if cancelled != test.cancelled {
				t.Fatalf(
					"cancelled = %v, want %v",
					cancelled,
					test.cancelled,
				)
			}
		})
	}
}

func TestSelectMultiplexerDefaultsToNone(t *testing.T) {
	selected, err := selectMultiplexer(
		t.Context(),
		"",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"selectMultiplexer() returned error: %v",
			err,
		)
	}

	if selected != multiplexerNone {
		t.Fatalf(
			"selected = %q, want %q",
			selected,
			multiplexerNone,
		)
	}
}

func TestSelectMultiplexerAcceptsDisabled(t *testing.T) {
	selected, err := selectMultiplexer(
		t.Context(),
		"disabled",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"selectMultiplexer() returned error: %v",
			err,
		)
	}

	if selected != multiplexerNone {
		t.Fatalf(
			"selected = %q, want %q",
			selected,
			multiplexerNone,
		)
	}
}
