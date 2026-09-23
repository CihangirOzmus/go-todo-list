package auth

import (
	"strings"
	"testing"
)

func TestPasswordRoundTrip(t *testing.T) {
	h, err := HashPassword("hunter22")
	if err != nil {
		t.Fatal(err)
	}
	if h == "hunter22" {
		t.Fatal("password stored in plaintext")
	}
	if err := CheckPassword(h, "hunter22"); err != nil {
		t.Errorf("expected match, got %v", err)
	}
	if err := CheckPassword(h, "wrong"); err == nil {
		t.Errorf("expected mismatch, got nil")
	}
}

// bcrypt refuses inputs longer than 72 bytes outright, so callers must treat
// length as a validation concern rather than an internal error.
func TestHashPassword_RejectsOverLongInput(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", MaxPasswordLen+1)); err == nil {
		t.Errorf("expected an error for a %d-byte password", MaxPasswordLen+1)
	}
	if _, err := HashPassword(strings.Repeat("a", MaxPasswordLen)); err != nil {
		t.Errorf("a %d-byte password should be accepted: %v", MaxPasswordLen, err)
	}
}

// Multi-byte characters count toward bcrypt's limit as bytes, not runes.
func TestHashPassword_LimitCountsBytesNotRunes(t *testing.T) {
	pw := strings.Repeat("é", MaxPasswordLen) // 2 bytes each
	if len(pw) <= MaxPasswordLen {
		t.Fatalf("test setup: %d bytes", len(pw))
	}
	if _, err := HashPassword(pw); err == nil {
		t.Error("expected an error: 144 bytes exceeds bcrypt's limit")
	}
}
