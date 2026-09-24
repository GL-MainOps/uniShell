package credentials

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreTokenEncryptsAndProtectsTokenFiles(t *testing.T) {
	root := t.TempDir()
	token := "private-token-value"

	if err := StoreToken(root, token); err != nil {
		t.Fatalf("StoreToken() returned error: %v", err)
	}

	for _, name := range []string{tokenFilename, keyFilename} {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) returned error: %v", name, err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Errorf("%s permissions = %04o, want 0600", name, got)
		}
	}

	ciphertext, err := os.ReadFile(filepath.Join(root, tokenFilename))
	if err != nil {
		t.Fatalf("ReadFile(.token) returned error: %v", err)
	}
	if bytes.Contains(ciphertext, []byte(token)) {
		t.Fatal("encrypted token file contains the plaintext token")
	}

	got, err := ReadStoredToken(root)
	if err != nil {
		t.Fatalf("ReadStoredToken() returned error: %v", err)
	}
	if got != token {
		t.Fatalf("ReadStoredToken() = %q, want %q", got, token)
	}
}

func TestReadStoredTokenRejectsModifiedCiphertext(t *testing.T) {
	root := t.TempDir()
	if err := StoreToken(root, "private-token-value"); err != nil {
		t.Fatalf("StoreToken() returned error: %v", err)
	}

	path := filepath.Join(root, tokenFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(.token) returned error: %v", err)
	}
	data[len(data)-1] ^= 0xff
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile(.token) returned error: %v", err)
	}

	if _, err := ReadStoredToken(root); err == nil {
		t.Fatal("ReadStoredToken() succeeded for modified ciphertext")
	}
}
