package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
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

func TestPayloadChangesDoNotFailTokenGate(t *testing.T) {
	encrypted, err := Encrypt(
		[]byte("uniShell test payload"),
		"test-password",
	)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	encrypted[len(encrypted)-1] ^= 0xff

	verified, err := Decrypt(encrypted, "test-password")
	if err != nil {
		t.Fatalf("Decrypt() returned error for a payload change: %v", err)
	}
	if verified[len(verified)-1] != encrypted[len(encrypted)-1] {
		t.Fatal("verified payload does not reflect the modified payload byte")
	}
}

func TestModifiedHeaderFailsAuthentication(t *testing.T) {
	encrypted, err := Encrypt(
		[]byte("uniShell test payload"),
		"test-password",
	)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	encrypted[14] ^= 0xff

	_, err = Decrypt(encrypted, "test-password")
	if err != credentials.ErrAuthenticationFailed {
		t.Fatalf(
			"Decrypt() error = %v, want %v",
			err,
			credentials.ErrAuthenticationFailed,
		)
	}
}

func TestUnsupportedVersionFails(t *testing.T) {
	encrypted, err := Encrypt(
		[]byte("uniShell test payload"),
		"test-password",
	)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	encrypted[4] = 1

	_, err = Decrypt(encrypted, "test-password")
	if err != ErrInvalidBundle {
		t.Fatalf(
			"Decrypt() error = %v, want %v",
			err,
			ErrInvalidBundle,
		)
	}
}

func TestTruncatedTokenGateHeaderFails(t *testing.T) {
	bundle, err := Encrypt([]byte("payload"), "test-password")
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	_, err = Decrypt(bundle[:macHeaderSize-1], "test-password")
	if err != ErrInvalidBundle {
		t.Fatalf("Decrypt() error = %v, want %v", err, ErrInvalidBundle)
	}
}

func TestValidParametersAcceptVersionTwoParameters(t *testing.T) {
	if !validParameters(
		argonTime,
		argonMemory,
		argonThreads,
	) {
		t.Fatal("v2 Argon2 parameters were rejected")
	}
}

func TestValidParametersAcceptLegacyVersionTwoParameters(t *testing.T) {
	if !validParameters(
		legacyArgonTime,
		legacyArgonMemory,
		legacyArgonThreads,
	) {
		t.Fatal("legacy v2 Argon2 parameters were rejected")
	}
}

func TestDecryptLegacyVersionTwoBundle(t *testing.T) {
	plaintext := []byte("uniShell legacy bundle payload")
	password := "legacy-password"

	salt := bytes.Repeat([]byte{0x42}, saltSize)

	key := deriveKey(
		[]byte(password),
		salt,
		legacyArgonTime,
		legacyArgonMemory,
		legacyArgonThreads,
	)
	defer zero(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher() returned error: %v", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("cipher.NewGCM() returned error: %v", err)
	}

	nonce := bytes.Repeat([]byte{0x24}, aead.NonceSize())

	bundle := Bundle{
		Version: legacyFormatVersion,
		Time:    legacyArgonTime,
		Memory:  legacyArgonMemory,
		Threads: legacyArgonThreads,
		Salt:    salt,
	}

	header, err := encodeHeader(bundle)
	if err != nil {
		t.Fatalf("encodeHeader() returned error: %v", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, header)

	bundle.Ciphertext = append(nonce, ciphertext...)

	encrypted, err := encodeBundle(bundle)
	if err != nil {
		t.Fatalf("encodeBundle() returned error: %v", err)
	}

	decrypted, err := Decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("Decrypt() returned error: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted legacy payload does not match plaintext")
	}
}

func TestBundleUsesVersionFourTokenGate(t *testing.T) {
	payload := []byte("uniShell test payload")
	bundle, err := Encrypt(payload, "test-password")
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	if bundle[4] != formatVersion {
		t.Fatalf("Version = %d, want %d", bundle[4], formatVersion)
	}
	if !bytes.Equal(bundle[5+macSize:], payload) {
		t.Fatal("version 4 bundle does not contain the plaintext payload")
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

func TestAuthenticationBundleIsDeterministic(t *testing.T) {
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

	if !bytes.Equal(first, second) {
		t.Fatal("same payload and token produced different authenticated bundles")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	plaintext := []byte("uniShell benchmark payload")
	password := "benchmark-password"

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := Encrypt(plaintext, password); err != nil {
			b.Fatalf("Encrypt() returned error: %v", err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	plaintext := []byte("uniShell benchmark payload")
	password := "benchmark-password"

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		b.Fatalf("Encrypt() returned error: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		decrypted, err := Decrypt(encrypted, password)
		if err != nil {
			b.Fatalf("Decrypt() returned error: %v", err)
		}

		if !bytes.Equal(decrypted, plaintext) {
			b.Fatal("decrypted payload does not match plaintext")
		}
	}
}
