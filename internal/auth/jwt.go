package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// di prod ini harus di ambil dari environment variable (.env)
var jwtSecretKey = []byte("ini_sangat_amat_rahasia_dan_panjang_123")

// isi data dari JWT
type CustomClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// fungsi utk membuat tiket masuk untuk user yg berhasil login
func GenerateJWT(username, role string) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &CustomClaims{
		username,
		role,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "secure-iam-api",
		},
	}

	// membuat token menggunakan algoritma HMAC SHA-256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// digital signature dgn secret key
	return token.SignedString(jwtSecretKey)
}

// fungsi untuk mengecek JWT token
func ValidateJWT(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// mitigasi khusus utk: 'alg: none' attack
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode penandatanganan tidak valid")
		}
		return jwtSecretKey, nil
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
