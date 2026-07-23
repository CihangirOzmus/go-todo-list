package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	Port        string
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        getEnv("PORT", "8080"),
	}
	ttlRaw := getEnv("JWT_TTL", "24h")
	d, err := time.ParseDuration(ttlRaw)
	if err != nil {
		return c, err
	}
	c.JWTTTL = d

	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return c, errors.New("JWT_SECRET is required")
	}
	return c, nil
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
