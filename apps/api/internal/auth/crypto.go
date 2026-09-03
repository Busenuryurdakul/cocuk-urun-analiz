package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func NewToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	return token, HashToken(token), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func NewNumericCode() (string, string, error) {
	raw := make([]byte, 3)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	n := int(raw[0])<<16 | int(raw[1])<<8 | int(raw[2])
	code := fmt.Sprintf("%06d", n%1000000)
	return code, HashToken(code), nil
}
