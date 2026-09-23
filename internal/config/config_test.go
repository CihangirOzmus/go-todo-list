package config

import (
	"testing"
	"time"
)

// setEnv puts the process env into a known-good baseline so each case only has
// to vary the one variable it cares about.
func setEnv(t *testing.T, databaseURL, jwtSecret, jwtTTL, port string) {
	t.Helper()
	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("JWT_SECRET", jwtSecret)
	t.Setenv("JWT_TTL", jwtTTL)
	t.Setenv("PORT", port)
}

func TestLoad_Defaults(t *testing.T) {
	setEnv(t, "postgres://localhost/db", "s3cret", "", "")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != "8080" {
		t.Errorf("port %q, want the 8080 default", c.Port)
	}
	if c.JWTTTL != 24*time.Hour {
		t.Errorf("ttl %v, want the 24h default", c.JWTTTL)
	}
	if c.DatabaseURL != "postgres://localhost/db" || c.JWTSecret != "s3cret" {
		t.Errorf("got %+v", c)
	}
}

func TestLoad_Overrides(t *testing.T) {
	setEnv(t, "postgres://db", "s", "90m", "3000")

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != "3000" {
		t.Errorf("port %q", c.Port)
	}
	if c.JWTTTL != 90*time.Minute {
		t.Errorf("ttl %v", c.JWTTTL)
	}
}

func TestLoad_RequiredVars(t *testing.T) {
	t.Run("missing DATABASE_URL", func(t *testing.T) {
		setEnv(t, "", "s3cret", "", "")
		if _, err := Load(); err == nil {
			t.Error("expected an error when DATABASE_URL is unset")
		}
	})
	t.Run("missing JWT_SECRET", func(t *testing.T) {
		setEnv(t, "postgres://db", "", "", "")
		if _, err := Load(); err == nil {
			t.Error("expected an error when JWT_SECRET is unset")
		}
	})
}

func TestLoad_CORSOrigins(t *testing.T) {
	t.Run("default covers Vite and CRA dev servers", func(t *testing.T) {
		setEnv(t, "postgres://db", "s3cret", "", "")
		t.Setenv("CORS_ORIGINS", "")

		c, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"http://localhost:5173", "http://localhost:3000"}
		if !equalStrings(c.CORSOrigins, want) {
			t.Errorf("origins %q, want %q", c.CORSOrigins, want)
		}
	})

	t.Run("comma-separated override", func(t *testing.T) {
		setEnv(t, "postgres://db", "s3cret", "", "")
		t.Setenv("CORS_ORIGINS", "https://todo.example.com,http://localhost:4200")

		c, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"https://todo.example.com", "http://localhost:4200"}
		if !equalStrings(c.CORSOrigins, want) {
			t.Errorf("origins %q, want %q", c.CORSOrigins, want)
		}
	})

	t.Run("wildcard passes through untouched", func(t *testing.T) {
		setEnv(t, "postgres://db", "s3cret", "", "")
		t.Setenv("CORS_ORIGINS", "*")

		c, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if !equalStrings(c.CORSOrigins, []string{"*"}) {
			t.Errorf("origins %q, want [*]", c.CORSOrigins)
		}
	})
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLoad_BadTTL(t *testing.T) {
	// A non-positive TTL would mint tokens that are already expired, and an
	// unparseable one is a straight typo — both must fail fast at startup.
	for _, ttl := range []string{"banana", "24", "0", "0s", "-1h"} {
		t.Run(ttl, func(t *testing.T) {
			setEnv(t, "postgres://db", "s3cret", ttl, "")
			if _, err := Load(); err == nil {
				t.Errorf("JWT_TTL=%q was accepted", ttl)
			}
		})
	}
}
