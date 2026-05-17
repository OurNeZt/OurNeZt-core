package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func NewSessionToken(byteLength int) (string, error) {
	if byteLength < 32 {
		byteLength = 32
	}

	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
