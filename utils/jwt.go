package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// PENTING: Di fase produksi, pindahkan secret key ini ke Environment Variable (.env)
var jwtKey = []byte("kunci_rahasia_super_aman_2026_kamu")

// Claims mendefinisikan data apa saja yang ingin kita masukkan ke dalam JWT
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateJWT untuk membuat token setelah user sukses login
func GenerateJWT(userID uint) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ValidateJWT untuk memverifikasi token yang dikirim oleh browser
func ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("token tidak valid")
	}

	return claims, nil
}
