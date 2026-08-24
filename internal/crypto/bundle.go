package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"

	"gitlab.com/mainops/uniShell/internal/credentials"
)

const (
	magic = "UNSB"

	formatVersion byte = 1

	saltSize = 16

	keySize = 32

	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
)

var (
	ErrInvalidBundle = errors.New("invalid encrypted bundle")
)

type Bundle struct {
	Version    byte
	Time       uint32
	Memory     uint32
	Threads    uint8
	Salt       []byte
	Ciphertext []byte
}

func Encrypt(plaintext []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, credentials.ErrEmptyToken
	}

	salt := make([]byte, saltSize)

	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate encryption salt: %w", err)
	}

	key := deriveKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
	)

	defer zero(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create encryption cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create authenticated cipher: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate encryption nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	payload := append(nonce, ciphertext...)

	return encodeBundle(Bundle{
		Version:    formatVersion,
		Time:       argonTime,
		Memory:     argonMemory,
		Threads:    argonThreads,
		Salt:       salt,
		Ciphertext: payload,
	})
}

func Decrypt(data []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, credentials.ErrEmptyToken
	}

	bundle, err := decodeBundle(data)
	if err != nil {
		return nil, err
	}

	key := deriveKey(
		[]byte(password),
		bundle.Salt,
		bundle.Time,
		bundle.Memory,
		bundle.Threads,
	)

	defer zero(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create decryption cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create authenticated cipher: %w", err)
	}

	if len(bundle.Ciphertext) < aead.NonceSize() {
		return nil, ErrInvalidBundle
	}

	nonce := bundle.Ciphertext[:aead.NonceSize()]
	ciphertext := bundle.Ciphertext[aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, credentials.ErrAuthenticationFailed
	}

	return plaintext, nil
}

func deriveKey(
	password []byte,
	salt []byte,
	time uint32,
	memory uint32,
	threads uint8,
) []byte {
	return argon2.IDKey(
		password,
		salt,
		time,
		memory,
		threads,
		keySize,
	)
}

func zero(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

func encodeBundle(bundle Bundle) ([]byte, error) {
	if len(bundle.Salt) != saltSize {
		return nil, ErrInvalidBundle
	}

	if len(bundle.Ciphertext) == 0 {
		return nil, ErrInvalidBundle
	}

	headerSize := 4 + 1 + 4 + 4 + 1 + saltSize

	result := make([]byte, headerSize+len(bundle.Ciphertext))

	offset := 0

	copy(result[offset:], magic)
	offset += 4

	result[offset] = bundle.Version
	offset++

	binary.BigEndian.PutUint32(result[offset:], bundle.Time)
	offset += 4

	binary.BigEndian.PutUint32(result[offset:], bundle.Memory)
	offset += 4

	result[offset] = bundle.Threads
	offset++

	copy(result[offset:], bundle.Salt)
	offset += saltSize

	copy(result[offset:], bundle.Ciphertext)

	return result, nil
}

func decodeBundle(data []byte) (Bundle, error) {
	const headerSize = 4 + 1 + 4 + 4 + 1 + saltSize

	if len(data) < headerSize {
		return Bundle{}, ErrInvalidBundle
	}

	offset := 0

	if string(data[offset:offset+4]) != magic {
		return Bundle{}, ErrInvalidBundle
	}

	offset += 4

	version := data[offset]
	offset++

	if version != formatVersion {
		return Bundle{}, ErrInvalidBundle
	}

	time := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	memory := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	threads := data[offset]
	offset++

	if time == 0 || memory == 0 || threads == 0 {
		return Bundle{}, ErrInvalidBundle
	}

	salt := make([]byte, saltSize)
	copy(salt, data[offset:offset+saltSize])
	offset += saltSize

	ciphertext := make([]byte, len(data)-offset)
	copy(ciphertext, data[offset:])

	return Bundle{
		Version:    version,
		Time:       time,
		Memory:     memory,
		Threads:    threads,
		Salt:       salt,
		Ciphertext: ciphertext,
	}, nil
}
