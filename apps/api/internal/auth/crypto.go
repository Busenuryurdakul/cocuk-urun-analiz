package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Version    = 19
	argon2MemoryKiB  = 65536
	argon2Iterations = 3
	argon2Parallel   = 4
	argon2SaltBytes  = 16
	argon2KeyBytes   = 32
)

// HashPassword returns a PHC-encoded Argon2id hash. Plaintext is never stored.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2MemoryKiB, argon2Parallel, argon2KeyBytes)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version, argon2MemoryKiB, argon2Iterations, argon2Parallel, saltB64, hashB64), nil
}

// CheckPassword verifies a password against an Argon2id PHC hash using constant-time comparison.
func CheckPassword(encoded, password string) bool {
	salt, expected, params, ok := decodeArgon2ID(encoded)
	if !ok {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallel, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

type argon2Params struct {
	memory     uint32
	iterations uint32
	parallel   uint8
}

func decodeArgon2ID(encoded string) (salt, hash []byte, params argon2Params, ok bool) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, argon2Params{}, false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2Version {
		return nil, nil, argon2Params{}, false
	}
	var m, t int
	var p int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return nil, nil, argon2Params{}, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, argon2Params{}, false
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, argon2Params{}, false
	}
	return salt, hash, argon2Params{
		memory:     uint32(m),
		iterations: uint32(t),
		parallel:   uint8(p),
	}, true
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
