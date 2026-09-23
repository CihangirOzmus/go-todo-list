package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go-todo-list/internal/models"
)

type Claims struct {
	UserID int64
	Role   models.Role
}

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

func (i *Issuer) Issue(userID int64, role models.Role) (string, error) {
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  strconv.FormatInt(userID, 10),
		"role": string(role),
		"iat":  now.Unix(),
		"exp":  now.Add(i.ttl).Unix(),
	})
	return tok.SignedString(i.secret)
}

func (i *Issuer) Parse(token string) (Claims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return i.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Claims{}, err
	}
	if !parsed.Valid {
		return Claims{}, errors.New("invalid token")
	}
	m, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, errors.New("invalid claims")
	}
	userID, err := parseSub(m["sub"])
	if err != nil {
		return Claims{}, err
	}
	role, _ := m["role"].(string)
	return Claims{UserID: userID, Role: models.Role(role)}, nil
}

// parseSub accepts the string form we now issue, as well as the numeric
// (float64) form emitted by older tokens, so tokens issued before this
// change keep parsing until they expire.
func parseSub(v any) (int64, error) {
	switch s := v.(type) {
	case string:
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, errors.New("invalid sub claim")
		}
		return id, nil
	case float64:
		return int64(s), nil
	default:
		return 0, errors.New("invalid sub claim")
	}
}
