package security

import (
	"crypto/rand"
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
