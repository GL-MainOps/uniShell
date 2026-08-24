package bundle

import (
	"errors"
	"os"
	"testing"

	"gitlab.com/mainops/uniShell/internal/credentials"
)

func TestTestBundleFixtureOpensWithTestToken(t *testing.T) {
	data, err := os.ReadFile("testdata/runtime.bundle")
	if err != nil {
		t.Fatalf("read test bundle fixture: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("test bundle fixture is empty")
	}

	archive, err := Open(data, testBundleToken)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}

	if len(archive) == 0 {
		t.Fatal("opened test bundle is empty")
	}
}

func TestTestBundleFixtureRejectsWrongToken(t *testing.T) {
	data, err := os.ReadFile("testdata/runtime.bundle")
	if err != nil {
		t.Fatalf("read test bundle fixture: %v", err)
	}

	_, err = Open(data, "wrong-test-token")
	if !errors.Is(err, credentials.ErrAuthenticationFailed) {
		t.Fatalf(
			"Open() error = %v, want %v",
			err,
			credentials.ErrAuthenticationFailed,
		)
	}
}

const testBundleToken = "test-fixture-token"
