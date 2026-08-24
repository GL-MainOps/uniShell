package credentials

import (
	"errors"
	"testing"
)

func TestErrAuthenticationFailedIsStable(t *testing.T) {
	err := ErrAuthenticationFailed

	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatal("ErrAuthenticationFailed should match itself")
	}
}
