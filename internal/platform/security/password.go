package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Params struct {
	MemoryKB    uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		MemoryKB:    64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// DummyPasswordHash supplies a valid verification target with the configured
// Argon2 cost, without running a KDF during startup. Its salt and key are public
// placeholders. A dummy verification must never authorize a login.
func DummyPasswordHash(params Argon2Params) string {
	if params.MemoryKB == 0 {
		params = DefaultArgon2Params()
	}
	return encodeArgon2idHash(params, make([]byte, params.SaltLength), make([]byte, params.KeyLength))
}

func HashPassword(password string, params Argon2Params) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", errors.New("password is required")
	}
	if params.MemoryKB == 0 {
		params = DefaultArgon2Params()
	}

	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKB, params.Parallelism, params.KeyLength)
	return encodeArgon2idHash(params, salt, key), nil
}

func encodeArgon2idHash(params Argon2Params, salt, key []byte) string {
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", params.MemoryKB, params.Iterations, params.Parallelism, encodedSalt, encodedKey)
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, expectedKey, err := decodeArgon2idHash(encodedHash)
	if err != nil {
		return false, err
	}

	actualKey := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKB, params.Parallelism, uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

func decodeArgon2idHash(encodedHash string) (Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, errors.New("invalid argon2id hash")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Argon2Params{}, nil, nil, errors.New("unsupported argon2id version")
	}

	params := Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.MemoryKB, &params.Iterations, &params.Parallelism); err != nil {
		return Argon2Params{}, nil, nil, errors.New("invalid argon2id parameters")
	}
	if params.MemoryKB == 0 || params.Iterations == 0 || params.Parallelism == 0 {
		return Argon2Params{}, nil, nil, errors.New("invalid argon2id parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}
	if len(salt) == 0 || len(key) == 0 {
		return Argon2Params{}, nil, nil, errors.New("invalid argon2id salt or key")
	}

	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(key))
	return params, salt, key, nil
}
