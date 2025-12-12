package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)



type Claims struct {
	mysecretkey string
}

func NewClaims(mysecretkey string) (*Claims, error) {
	if mysecretkey == "" {
		fmt.Errorf("MYSECRETKEY is not set")
	}
	return &Claims{mysecretkey: mysecretkey}, nil
}

func (c *Claims) NewAccessToken(userID int64, TTL time.Duration) (string, error) {
	accessExpiration := time.Now().Add(TTL)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     accessExpiration.Unix(),
		"iat":     time.Now().Unix(),
	})
	accessString, err := accessToken.SignedString([]byte(c.mysecretkey))
	if err != nil {
		return "", err
	}
	return accessString, nil
}

func (c *Claims) NewRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (c *Claims) Parse(accessToken string) (int64, error) {

	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.mysecretkey), nil
	})

	if err != nil {
		return 0, fmt.Errorf("parse token:%w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	sub, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}
	return int64(sub), nil
}
