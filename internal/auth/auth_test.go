package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("expected correct password to verify")
	}
	if CheckPassword(hash, "wrong password") {
		t.Fatal("expected wrong password to fail verification")
	}
}

func TestHashPassword_StoresNoPlaintext(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "hunter2" {
		t.Fatal("hash must not equal the plaintext password")
	}
}

func TestNewSessionToken_UniqueAndNonEmpty(t *testing.T) {
	a, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken: %v", err)
	}
	b, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken: %v", err)
	}
	if a == "" || b == "" {
		t.Fatal("expected non-empty tokens")
	}
	if a == b {
		t.Fatal("expected two calls to produce distinct tokens")
	}
}
