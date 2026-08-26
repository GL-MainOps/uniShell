package shellargs

import (
	"reflect"
	"testing"
)

func TestTokenizeEmpty(t *testing.T) {
	got, err := Tokenize("")
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf(
			"Tokenize() = %#v, want empty slice",
			got,
		)
	}
}

func TestTokenizeWhitespaceSeparatedArguments(t *testing.T) {
	got, err := Tokenize(
		"-f /tmp/tmux.conf -L work",
	)
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	want := []string{
		"-f",
		"/tmp/tmux.conf",
		"-L",
		"work",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Tokenize() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestTokenizeSingleQuotes(t *testing.T) {
	got, err := Tokenize(
		`-L 'work session'`,
	)
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	want := []string{
		"-L",
		"work session",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Tokenize() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestTokenizeDoubleQuotes(t *testing.T) {
	got, err := Tokenize(
		`--layout "compact layout.kdl"`,
	)
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	want := []string{
		"--layout",
		"compact layout.kdl",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Tokenize() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestTokenizeEscapedCharacters(t *testing.T) {
	got, err := Tokenize(
		`--name work\ session`,
	)
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	want := []string{
		"--name",
		"work session",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Tokenize() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestTokenizeEmptyQuotedArguments(t *testing.T) {
	got, err := Tokenize(
		`"" ''`,
	)
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	want := []string{
		"",
		"",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Tokenize() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestTokenizeMixedQuotedAndUnquotedContent(t *testing.T) {
	got, err := Tokenize(
		`prefix"quoted value"'tail'`,
	)
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	want := []string{
		"prefixquoted valuetail",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Tokenize() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestTokenizePreservesShellSyntaxAsLiteralText(t *testing.T) {
	got, err := Tokenize(
		`$(id) "foo; bar" foo|bar`,
	)
	if err != nil {
		t.Fatalf("Tokenize() returned error: %v", err)
	}

	want := []string{
		"$(id)",
		"foo; bar",
		"foo|bar",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"Tokenize() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestTokenizeUnterminatedSingleQuote(t *testing.T) {
	_, err := Tokenize(
		`--name 'work`,
	)
	if err == nil {
		t.Fatal(
			"Tokenize() returned nil error, want unterminated quote error",
		)
	}
}

func TestTokenizeUnterminatedDoubleQuote(t *testing.T) {
	_, err := Tokenize(
		`--name "work`,
	)
	if err == nil {
		t.Fatal(
			"Tokenize() returned nil error, want unterminated quote error",
		)
	}
}

func TestTokenizeTrailingEscape(t *testing.T) {
	_, err := Tokenize(
		`--name work\`,
	)
	if err == nil {
		t.Fatal(
			"Tokenize() returned nil error, want trailing escape error",
		)
	}
}

func TestTokenizeTrailingEscapeInsideDoubleQuotes(t *testing.T) {
	_, err := Tokenize(
		`--name "work\`,
	)
	if err == nil {
		t.Fatal(
			"Tokenize() returned nil error, want trailing escape error",
		)
	}
}
