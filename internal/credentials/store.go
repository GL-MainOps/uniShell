package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	tokenFilename = ".token"
	keyFilename   = ".token.key"
)

var tokenFileHeader = []byte("UNISHELL-TOKEN-V1\n")

// ReadStoredToken decrypts the locally saved token. Both the ciphertext and
// its per-install key are private to the current OS account. This avoids
// storing the token in plaintext, while the directory permissions remain the
// actual boundary against other accounts on the machine.
func ReadStoredToken(runtimeRoot string) (string, error) {
	key, err := readPrivateKey(filepath.Join(runtimeRoot, keyFilename))
	if err != nil {
		return "", err
	}

	path := filepath.Join(runtimeRoot, tokenFilename)
	if err := requireRegularPrivateFile(path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read stored authentication token: %w", err)
	}
	if len(data) < len(tokenFileHeader) || string(data[:len(tokenFileHeader)]) != string(tokenFileHeader) {
		return "", errors.New("stored authentication token has an unsupported format")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("initialize stored-token cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("initialize stored-token authentication: %w", err)
	}
	payload := data[len(tokenFileHeader):]
	if len(payload) < aead.NonceSize() {
		return "", errors.New("stored authentication token is truncated")
	}
	nonce, ciphertext := payload[:aead.NonceSize()], payload[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, tokenFileHeader)
	if err != nil {
		return "", errors.New("stored authentication token could not be decrypted")
	}
	if len(plaintext) == 0 {
		return "", ErrEmptyToken
	}
	return string(plaintext), nil
}

// StoreToken encrypts the token using AES-GCM and writes it with owner-only
// permissions beneath the persistent runtime root.
func StoreToken(runtimeRoot, token string) error {
	if token == "" {
		return ErrEmptyToken
	}
	if err := os.MkdirAll(runtimeRoot, 0700); err != nil {
		return fmt.Errorf("create token storage directory: %w", err)
	}
	if err := os.Chmod(runtimeRoot, 0700); err != nil {
		return fmt.Errorf("protect token storage directory: %w", err)
	}

	keyPath := filepath.Join(runtimeRoot, keyFilename)
	key, err := readPrivateKey(keyPath)
	if errors.Is(err, os.ErrNotExist) {
		key, err = createPrivateKey(keyPath)
	}
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("initialize stored-token cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("initialize stored-token authentication: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate token encryption nonce: %w", err)
	}
	data := append([]byte(nil), tokenFileHeader...)
	data = append(data, nonce...)
	data = aead.Seal(data, nonce, []byte(token), tokenFileHeader)
	return writePrivateFile(filepath.Join(runtimeRoot, tokenFilename), data)
}

func readPrivateKey(path string) ([]byte, error) {
	if err := requireRegularPrivateFile(path); err != nil {
		return nil, err
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("token encryption key has an invalid size")
	}
	return key, nil
}

func createPrivateKey(path string) ([]byte, error) {
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".token-key-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create token encryption key: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("generate token encryption key: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("protect token encryption key: %w", err)
	}
	if _, err := file.Write(key); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("write token encryption key: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("sync token encryption key: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close token encryption key: %w", err)
	}
	if err := os.Link(tempPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return readPrivateKey(path)
		}
		return nil, fmt.Errorf("publish token encryption key: %w", err)
	}
	return key, nil
}

func writePrivateFile(path string, data []byte) error {
	directory := filepath.Dir(path)
	file, err := os.CreateTemp(directory, ".token-*.tmp")
	if err != nil {
		return fmt.Errorf("create encrypted token file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return fmt.Errorf("protect encrypted token file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write encrypted token file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync encrypted token file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close encrypted token file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish encrypted token file: %w", err)
	}
	return nil
}

func requireRegularPrivateFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("token storage path %q is not a regular file", path)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("protect token storage file: %w", err)
	}
	return nil
}
