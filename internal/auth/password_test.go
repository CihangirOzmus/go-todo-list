package auth

import "testing"

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
