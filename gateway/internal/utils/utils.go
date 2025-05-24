package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kerucko/diploma/gateway/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthManager struct {
	secret    string
	expiresIn time.Duration
}

func NewAuthManager(secret string, expiresIn time.Duration) *AuthManager {
	return &AuthManager{
		secret:    secret,
		expiresIn: expiresIn,
	}
}

func (a *AuthManager) GenerateToken(user *models.User) (string, error) {
	claims := models.Claims{
		UserID: user.ID,
		Role:   user.Role,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secret))
}

func (a *AuthManager) ParseToken(tokenStr string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*models.Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrInvalidKey
	}

	return claims, nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
