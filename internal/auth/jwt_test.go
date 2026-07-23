package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go-todo-list/internal/models"
)

func TestIssueAndParse(t *testing.T) {
	iss := NewIssuer("secret", time.Hour)
	tok, err := iss.Issue(42, models.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	c, err := iss.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != 42 || c.Role != models.RoleAdmin {
		t.Errorf("got %+v", c)
	}
}

// TestParseLegacyNumericSub verifies tokens minted before sub was switched to
// a string (i.e. a numeric JSON sub) still parse.
func TestParseLegacyNumericSub(t *testing.T) {
	iss := NewIssuer("secret", time.Hour)
	now := time.Now()
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  42, // numeric, as older tokens encoded it
		"role": string(models.RoleAdmin),
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour).Unix(),
	})
	tok, err := legacy.SignedString(iss.secret)
	if err != nil {
		t.Fatal(err)
	}
	c, err := iss.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != 42 || c.Role != models.RoleAdmin {
		t.Errorf("got %+v", c)
	}
}

func TestParseExpired(t *testing.T) {
	iss := NewIssuer("secret", -time.Minute)
	tok, err := iss.Issue(1, models.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := iss.Parse(tok); err == nil {
		t.Errorf("expected error for expired token")
	}
}

func TestParseWrongSecret(t *testing.T) {
	iss := NewIssuer("secret", time.Hour)
	tok, _ := iss.Issue(1, models.RoleUser)
	iss2 := NewIssuer("different", time.Hour)
	if _, err := iss2.Parse(tok); err == nil {
		t.Errorf("expected error when parsing with wrong secret")
	}
}

func TestParseTampered(t *testing.T) {
	iss := NewIssuer("secret", time.Hour)
	tok, _ := iss.Issue(1, models.RoleUser)
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	// Flip a character in the payload segment.
	parts[1] = parts[1] + "x"
	if _, err := iss.Parse(strings.Join(parts, ".")); err == nil {
		t.Errorf("expected error for tampered token")
	}
}
