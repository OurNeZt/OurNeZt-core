package security

import (
	"strings"
	"testing"
)

func fastArgon2Params() Argon2Params {
	return Argon2Params{
		MemoryKB:    1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func TestHashPasswordAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple", fastArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("hash prefix = %q, want argon2id prefix", hash)
	}
	if strings.Contains(hash, "correct horse battery staple") {
		t.Fatal("hash contains the raw password")
	}

	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword = false, want true")
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("right-password", fastArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword = true, want false")
	}
}

func TestHashPasswordRejectsBlankPassword(t *testing.T) {
	if _, err := HashPassword("   ", fastArgon2Params()); err == nil {
		t.Fatal("HashPassword returned nil error for blank password")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	ok, err := VerifyPassword("password", "not-a-real-hash")
	if err == nil {
		t.Fatal("VerifyPassword returned nil error for malformed hash")
	}
	if ok {
		t.Fatal("VerifyPassword = true, want false")
	}
}

