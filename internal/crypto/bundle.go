package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"

	"gitlab.com/mainops/uniShell/internal/credentials"
)

const (
	magic = "UNSB"

	legacyFormatVersion byte = 2
	integrityVersion    byte = 3
	formatVersion       byte = 4
	authenticationLabel      = "uniShell token gate"

	macSize       = sha256.Size
	macHeaderSize = 5 + macSize

	saltSize = 16
	keySize  = 32

	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4

	legacyArgonTime    uint32 = 1
	legacyArgonMemory  uint32 = 8 * 1024
	legacyArgonThreads uint8  = 1
)

var ErrInvalidBundle = errors.New("invalid runtime bundle")

type Bundle struct {
	Version    byte
	Time       uint32
	Memory     uint32
	Threads    uint8
	Salt       []byte
	Ciphertext []byte
}

// Authenticate creates a version 4 bundle containing the payload in plaintext
// and a token-gate HMAC-SHA256 tag. The tag covers only a fixed challenge, not
// the payload. This is an application gate, not payload integrity protection.
func Authenticate(payload []byte, token string) ([]byte, error) {
	if token == "" {
		return nil, credentials.ErrEmptyToken
	}

	prefix := make([]byte, 5)
	copy(prefix, magic)
	prefix[4] = formatVersion

	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write(prefix)
	_, _ = mac.Write([]byte(authenticationLabel))
	tag := mac.Sum(nil)

	result := make([]byte, len(prefix)+len(tag)+len(payload))
	copy(result, prefix)
	copy(result[len(prefix):], tag)
	copy(result[len(prefix)+len(tag):], payload)
	return result, nil
}

// Verify checks the token gate and returns the bundle payload as a read-only
// view into data. Version 3 integrity-authenticated and version 2 encrypted
// bundles remain readable for compatibility.
func Verify(data []byte, token string) ([]byte, error) {
	if token == "" {
		return nil, credentials.ErrEmptyToken
	}
	if len(data) < 5 || string(data[:4]) != magic {
		return nil, ErrInvalidBundle
	}

	switch data[4] {
	case formatVersion:
		return verifyV4(data, token)
	case integrityVersion:
		return verifyV3(data, token)
	case legacyFormatVersion:
		return decryptV2(data, token)
	default:
		return nil, ErrInvalidBundle
	}
}

// Encrypt is retained as a source-compatible alias. New bundles are
// authenticated but are not encrypted.
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	return Authenticate(plaintext, password)
}

// Decrypt is retained as a source-compatible alias for Verify.
func Decrypt(data []byte, password string) ([]byte, error) {
	return Verify(data, password)
}

func verifyV4(data []byte, token string) ([]byte, error) {
	const prefixSize = 5
	if len(data) < macHeaderSize {
		return nil, ErrInvalidBundle
	}

	prefix := data[:prefixSize]
	tag := data[prefixSize:macHeaderSize]
	payload := data[macHeaderSize:]

	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write(prefix)
	_, _ = mac.Write([]byte(authenticationLabel))
	if !hmac.Equal(tag, mac.Sum(nil)) {
		return nil, credentials.ErrAuthenticationFailed
	}

	return payload, nil
}

func verifyV3(data []byte, token string) ([]byte, error) {
	const prefixSize = 5
	if len(data) < macHeaderSize {
		return nil, ErrInvalidBundle
	}

	prefix := data[:prefixSize]
	tag := data[prefixSize:macHeaderSize]
	payload := data[macHeaderSize:]

	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write(prefix)
	_, _ = mac.Write(payload)
	if !hmac.Equal(tag, mac.Sum(nil)) {
		return nil, credentials.ErrAuthenticationFailed
	}
	return payload, nil
}

func decryptV2(data []byte, token string) ([]byte, error) {
	bundle, err := decodeBundle(data)
	if err != nil {
		return nil, err
	}

	key := deriveKey(
		[]byte(token),
		bundle.Salt,
		bundle.Time,
		bundle.Memory,
		bundle.Threads,
	)
	defer zero(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create legacy decryption cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create legacy authenticated cipher: %w", err)
	}
	if len(bundle.Ciphertext) < aead.NonceSize() {
		return nil, ErrInvalidBundle
	}

	nonce := bundle.Ciphertext[:aead.NonceSize()]
	ciphertext := bundle.Ciphertext[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, data[:bundleHeaderSize])
	if err != nil {
		return nil, credentials.ErrAuthenticationFailed
	}
	return plaintext, nil
}

func deriveKey(password []byte, salt []byte, time uint32, memory uint32, threads uint8) []byte {
	return argon2.IDKey(password, salt, time, memory, threads, keySize)
}

func zero(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

func validParameters(time uint32, memory uint32, threads uint8) bool {
	return time == argonTime && memory == argonMemory && threads == argonThreads ||
		time == legacyArgonTime && memory == legacyArgonMemory && threads == legacyArgonThreads
}

func encodeHeader(bundle Bundle) ([]byte, error) {
	if bundle.Version != legacyFormatVersion || !validParameters(bundle.Time, bundle.Memory, bundle.Threads) || len(bundle.Salt) != saltSize {
		return nil, ErrInvalidBundle
	}

	const headerSize = 4 + 1 + 4 + 4 + 1 + saltSize
	header := make([]byte, headerSize)
	offset := 0
	copy(header[offset:], magic)
	offset += 4
	header[offset] = bundle.Version
	offset++
	binary.BigEndian.PutUint32(header[offset:], bundle.Time)
	offset += 4
	binary.BigEndian.PutUint32(header[offset:], bundle.Memory)
	offset += 4
	header[offset] = bundle.Threads
	offset++
	copy(header[offset:], bundle.Salt)
	return header, nil
}

func encodeBundle(bundle Bundle) ([]byte, error) {
	header, err := encodeHeader(bundle)
	if err != nil {
		return nil, err
	}
	if len(bundle.Ciphertext) == 0 {
		return nil, ErrInvalidBundle
	}
	result := make([]byte, len(header)+len(bundle.Ciphertext))
	copy(result, header)
	copy(result[len(header):], bundle.Ciphertext)
	return result, nil
}

const bundleHeaderSize = 4 + 1 + 4 + 4 + 1 + saltSize

func decodeBundle(data []byte) (Bundle, error) {
	if len(data) < bundleHeaderSize || string(data[:4]) != magic || data[4] != legacyFormatVersion {
		return Bundle{}, ErrInvalidBundle
	}

	offset := 5
	time := binary.BigEndian.Uint32(data[offset:])
	offset += 4
	memory := binary.BigEndian.Uint32(data[offset:])
	offset += 4
	threads := data[offset]
	offset++
	if !validParameters(time, memory, threads) {
		return Bundle{}, ErrInvalidBundle
	}

	salt := append([]byte(nil), data[offset:offset+saltSize]...)
	offset += saltSize
	ciphertext := append([]byte(nil), data[offset:]...)
	return Bundle{
		Version:    legacyFormatVersion,
		Time:       time,
		Memory:     memory,
		Threads:    threads,
		Salt:       salt,
		Ciphertext: ciphertext,
	}, nil
}
