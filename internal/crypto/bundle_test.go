package crypto

import (
	"bytes"
	"testing"

	"gitlab.com/mainops/uniShell/internal/credentials"
)

func TestEncryptDecrypt(t *testing.T) {
	plaintext := []byte("uniShell test payload")
	password := "test-password"

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt() returned error: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted payload does not match plaintext")
	}
}

func TestWrongPasswordFailsAuthentication(t *testing.T) {
	plaintext := []byte("uniShell test payload")

	encrypted, err := Encrypt(plaintext, "test-password")
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	_, err = Decrypt(encrypted, "wrong-password")
	if err != credentials.ErrAuthenticationFailed {
		t.Fatalf(
			"Decrypt() error = %v, want %v",
			err,
			credentials.ErrAuthenticationFailed,
		)
	}
}

func TestModifiedBundleFailsAuthentication(t *testing.T) {
	encrypted, err := Encrypt(
		[]byte("uniShell test payload"),
		"test-password",
	)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	encrypted[len(encrypted)-1] ^= 0xff

	_, err = Decrypt(encrypted, "test-password")
	if err != credentials.ErrAuthenticationFailed {
		t.Fatalf(
			"Decrypt() error = %v, want %v",
			err,
			credentials.ErrAuthenticationFailed,
		)
	}
}

func TestInvalidBundleFails(t *testing.T) {
	_, err := Decrypt(
		[]byte("not a uniShell bundle"),
		"test-password",
	)

	if err != ErrInvalidBundle {
		t.Fatalf(
			"Decrypt() error = %v, want %v",
			err,
			ErrInvalidBundle,
		)
	}
}

func TestEmptyPasswordFails(t *testing.T) {
	_, err := Encrypt([]byte("payload"), "")

	if err != credentials.ErrEmptyToken {
		t.Fatalf(
			"Encrypt() error = %v, want %v",
			err,
			credentials.ErrEmptyToken,
		)
	}
}

func TestEncryptionProducesDifferentBundles(t *testing.T) {
	plaintext := []byte("uniShell test payload")
	password := "test-password"

	first, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("first Encrypt() returned error: %v", err)
	}

	second, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("second Encrypt() returned error: %v", err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("two encryptions produced identical bundles")
	}
}
