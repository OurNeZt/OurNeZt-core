package security

import "testing"

func TestDummyPasswordHashUsesConfiguredCost(t *testing.T) {
	for _, configured := range []Argon2Params{{}, fastArgon2Params()} {
		want := configured
		if want.MemoryKB == 0 {
			want = DefaultArgon2Params()
		}
		hash := DummyPasswordHash(configured)
		got, salt, key, err := decodeArgon2idHash(hash)
		if err != nil || got != want || len(salt) != int(want.SaltLength) || len(key) != int(want.KeyLength) {
			t.Fatalf("invalid dummy verification target: params=%+v err=%v", got, err)
		}
	}
	for _, password := range []string{"password", "", " "} {
		ok, err := VerifyPassword(password, DummyPasswordHash(fastArgon2Params()))
		if err != nil || ok {
			t.Fatalf("dummy verification: matched=%v err=%v", ok, err)
		}
	}
}

func TestVerifyPasswordRejectsInvalidStoredParameters(t *testing.T) {
	for _, hash := range []string{
		"$argon2id$v=19$m=0,t=1,p=1$YWJj$ZGVm",
		"$argon2id$v=19$m=1024,t=0,p=1$YWJj$ZGVm",
		"$argon2id$v=19$m=1024,t=1,p=0$YWJj$ZGVm",
		"$argon2id$v=19$m=1024,t=1,p=1$$ZGVm",
		"$argon2id$v=19$m=1024,t=1,p=1$YWJj$",
	} {
		ok, err := VerifyPassword("password", hash)
		if err == nil || ok {
			t.Fatalf("invalid stored hash authenticated: matched=%v err=%v", ok, err)
		}
	}
}
