package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type AccessClaims struct {
	UserID         string `json:"uid"`
	SessionID      string `json:"sid"`
	OrganizationID string `json:"oid"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret    []byte
	accessTTL time.Duration
}

func NewJWTManager(secret string, accessTTL time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), accessTTL: accessTTL}
}

func (m *JWTManager) IssueAccessToken(userID, sessionID, organizationID primitive.ObjectID) (string, time.Time, error) {
	now := time.Now().UTC()
	expires := now.Add(m.accessTTL)
	claims := AccessClaims{
		UserID:         userID.Hex(),
		SessionID:      sessionID.Hex(),
		OrganizationID: organizationID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
			Subject:   userID.Hex(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expires, nil
}

func (m *JWTManager) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidAccessToken
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidAccessToken
	}
	return claims, nil
}
