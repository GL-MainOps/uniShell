package credentials

import "testing"

func TestResolveUsesEnvironmentToken(t *testing.T) {
	t.Setenv(environmentVariable, "environment-token")

	token, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if token != "environment-token" {
		t.Fatalf("token = %q, want %q", token, "environment-token")
	}
}

func TestValidateRejectsEmptyToken(t *testing.T) {
	_, err := validate("")

	if err != ErrEmptyToken {
		t.Fatalf("validate() error = %v, want %v", err, ErrEmptyToken)
	}
}

func TestValidateAcceptsToken(t *testing.T) {
	token, err := validate("test-token")
	if err != nil {
		t.Fatalf("validate() returned error: %v", err)
	}

	if token != "test-token" {
		t.Fatalf("token = %q, want %q", token, "test-token")
	}
}
