package auth

import (
	"testing"
)

func TestResolveUsesExplicitToken(t *testing.T) {
	t.Setenv(environmentVariable, "environment-token")

	token, err := Resolve(Options{
		Explicit: "explicit-token",
	})
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if token != "explicit-token" {
		t.Fatalf("token = %q, want %q", token, "explicit-token")
	}
}

func TestResolveUsesEnvironmentToken(t *testing.T) {
	t.Setenv(environmentVariable, "environment-token")

	token, err := Resolve(Options{})
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if token != "environment-token" {
		t.Fatalf("token = %q, want %q", token, "environment-token")
	}
}

func TestResolveRejectsEmptyExplicitToken(t *testing.T) {
	t.Setenv(environmentVariable, "environment-token")

	// An empty explicit value means "not supplied", so the environment
	// variable remains the next source in the precedence chain.
	token, err := Resolve(Options{
		Explicit: "",
	})
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
