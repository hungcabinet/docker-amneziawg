package auth

import (
	"testing"
	"time"
)

func TestEncodeDecode(t *testing.T) {
	t.Parallel()
	secret := []byte("0123456789abcdef0123456789abcdef")
	value, err := Encode(secret, Session{Username: "admin", Expiry: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	got, err := Decode(secret, value)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Username != "admin" {
		t.Errorf("Username = %q", got.Username)
	}
}

func TestDecode_RejectsTamperAndExpiry(t *testing.T) {
	t.Parallel()
	secret := []byte("0123456789abcdef0123456789abcdef")
	value, err := Encode(secret, Session{Username: "admin", Expiry: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if _, err := Decode(secret, value+"x"); err == nil {
		t.Fatal("tampered cookie accepted")
	}
	if _, err := Decode([]byte("other-secret-other-secret-other!"), value); err == nil {
		t.Fatal("wrong secret accepted")
	}
	expired, err := Encode(secret, Session{Username: "admin", Expiry: time.Now().Add(-time.Minute)})
	if err != nil {
		t.Fatalf("Encode() expired error = %v", err)
	}
	if _, err := Decode(secret, expired); err == nil {
		t.Fatal("expired cookie accepted")
	}
}
