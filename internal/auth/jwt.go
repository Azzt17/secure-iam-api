package auth

import (
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
