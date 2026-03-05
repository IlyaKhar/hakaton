package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims описывает payload JWT.
// Пока минимально: только ID пользователя.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken генерирует access-токен для пользователя.
// TODO: передавать сюда секрет и время жизни из конфига.
func GenerateToken(userID string, secret string, ttl time.Duration) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

// ParseToken парсит и валидирует JWT и возвращает ID пользователя.
// TODO: дописать проверки (issuer, audience и т.п. при необходимости).
func ParseToken(tokenString string, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}

	return claims.UserID, nil
}
