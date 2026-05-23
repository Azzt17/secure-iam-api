package auth

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// isi data dari JWT
type CustomClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// fungsi utk membuat tiket masuk untuk user yg berhasil login
func GenerateJWT(username, role string) (string, error) {
	// ambil secret key dari env secara dinamis
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("FATAL: JWT_SECRET tidak dikonfigurasi di environment!")
	}

	claims := &CustomClaims{
		username,
		role,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// membuat token menggunakan algoritma HMAC SHA-256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// digital signature dgn secret key
	return token.SignedString([]byte(secret))
}

// fungsi untuk mengecek JWT token
func ValidateJWT(tokenString string) (*CustomClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("server misconfiguration: missing jwt secret")
	}
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// mitigasi khusus utk: 'alg: none' attack
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode penandatanganan tidak valid")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	// pastikan token blm expired dan claims bisa diekstrak
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token tidak valid")
}
