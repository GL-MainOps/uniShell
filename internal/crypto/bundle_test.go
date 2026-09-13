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

func TestInvalidKDFParametersFail(t *testing.T) {
	encrypted, err := Encrypt(
		[]byte("uniShell test payload"),
		"test-password",
	)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	// The Argon2 memory field occupies bytes 9-12.
	// Change 65536 KiB to 65537 KiB.
	encrypted[12] = 1

	_, err = Decrypt(encrypted, "test-password")
	if err != ErrInvalidBundle {
		t.Fatalf(
			"Decrypt() error = %v, want %v",
			err,
			ErrInvalidBundle,
		)
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
		Version: formatVersion,
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

func TestBundleUsesVersionTwoParameters(t *testing.T) {
	encrypted, err := Encrypt(
		[]byte("uniShell test payload"),
		"test-password",
	)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	bundle, err := decodeBundle(encrypted)
	if err != nil {
		t.Fatalf("decodeBundle() returned error: %v", err)
	}

	if bundle.Version != 2 {
		t.Fatalf("Version = %d, want 2", bundle.Version)
	}

	if bundle.Time != 1 {
		t.Fatalf("Time = %d, want 1", bundle.Time)
	}

	if bundle.Memory != 64*1024 {
		t.Fatalf("Memory = %d, want %d", bundle.Memory, 64*1024)
	}

	if bundle.Threads != 4 {
		t.Fatalf("Threads = %d, want 4", bundle.Threads)
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
