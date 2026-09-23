package auth

import "golang.org/x/crypto/bcrypt"

// MaxPasswordLen is bcrypt's hard input limit. GenerateFromPassword rejects
// longer passwords with an error rather than truncating them, so callers must
// validate length up front instead of treating it as an internal failure.
const MaxPasswordLen = 72

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}
