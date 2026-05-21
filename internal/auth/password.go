package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// fungsi eknripsi ke hash bcrypt
func HashPassword(password string) (string, error) {
	// menggunakan cost factor 12:
	// semakin tinggi angkanya, semakin lambat prosesnya,
	// agar memperlambar brute-force
	// defaultnya 10, 12 utk standar kemanan modern
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// fungsi utk mengecek ketika user login
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
