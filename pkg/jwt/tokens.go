package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	UserID int64
	Exp    int64
	
}

type Manager struct {
	signingKey []byte
}

func NewManager(signingKey string) (*Manager, error) {
	if signingKey == "" {
		return nil, fmt.Errorf("empty signing key")
	}
	return &Manager{signingKey: []byte(signingKey)}, nil
}

func (m *Manager) NewAccessToken(userID int64, ttl time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(ttl).Unix(),
		"iat": time.Now().Unix(),
	})

	return token.SignedString(m.signingKey)
}

func (m *Manager) NewRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *Manager) Parse(accessToken string) (*TokenClaims, error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.signingKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return nil, fmt.Errorf("token does not contain sub (user_id)")
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("token does not contain exp")
	}

	return &TokenClaims{
		UserID: int64(subFloat),
		Exp:    int64(expFloat),
	}, nil
}
