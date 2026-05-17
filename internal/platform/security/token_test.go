package security

import (
	"strings"
	"testing"
)

func TestNewSessionTokenUsesMinimumEntropy(t *testing.T) {
	token, err := NewSessionToken(8)
	if err != nil {
		t.Fatalf("NewSessionToken returned error: %v", err)
	}

	if len(token) != 43 {
		t.Fatalf("len(token) = %d, want 43 for 32 random bytes", len(token))
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("token %q is not raw URL-safe base64", token)
	}
}

func TestNewSessionTokenReturnsDifferentValues(t *testing.T) {
	first, err := NewSessionToken(32)
	if err != nil {
		t.Fatalf("NewSessionToken first returned error: %v", err)
	}
	second, err := NewSessionToken(32)
	if err != nil {
		t.Fatalf("NewSessionToken second returned error: %v", err)
	}
	if first == second {
		t.Fatal("two session tokens were identical")
	}
}

func TestHashTokenDeterministicAndDistinct(t *testing.T) {
	first := HashToken("token-one")
	second := HashToken("token-one")
	third := HashToken("token-two")

	if first != second {
		t.Fatalf("HashToken not deterministic: %q != %q", first, second)
	}
	if first == third {
		t.Fatalf("HashToken collision for different inputs: %q", first)
	}
	if strings.ContainsAny(first, "+/=") {
		t.Fatalf("hash %q is not raw URL-safe base64", first)
	}
}
