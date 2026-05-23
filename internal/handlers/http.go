package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"secure-iam-api/internal/auth"
	"secure-iam-api/internal/domain"
	"secure-iam-api/internal/middleware"
	"secure-iam-api/internal/service"

	"github.com/go-playground/validator/v10"
)

type IAMHandler struct {
	service  service.IAMService
	validate *validator.Validate
}

func NewIAMHandler(service service.IAMService, validate *validator.Validate) *IAMHandler {
	return &IAMHandler{
		service:  service,
		validate: validate,
	}
}

// Bantuan format error (menggunakan comma-ok idiom)
func formatValidationError(err error) map[string]string {
	errorsMap := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, e := range ve {
			errorsMap[e.Field()] = fmt.Sprintf("Gagal pada aturan validasi: %s", e.Tag())
		}
	} else {
		errorsMap["general"] = "Terjadi kesalahan pada validasi data"
		log.Printf("Non-validation error caught: %v", err)
	}
	return errorsMap
}

// Struktur Request
type RegisterRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=3,max=32"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=3,max=32"`
	Password string `json:"password" validate:"required"`
}

func (h *IAMHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Validasi Gagal", "details": formatValidationError(err)})
		return
	}

	// Pendelegasian murni ke Service Layer
	err := h.service.RegisterUser(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			http.Error(w, "Conflict: Username sudah terdaftar", http.StatusConflict)
			return
		}
		log.Printf("Register error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "User berhasil didaftarkan"})
}

func (h *IAMHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Validasi Gagal", "details": formatValidationError(err)})
		return
	}

	// Pendelegasian ke Service Layer
	tokenString, err := h.service.LoginUser(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "Kredensial tidak valid", http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenString,
		Expires:  time.Now().Add(1 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Login berhasil"})
}

func (h *IAMHandler) DeductWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(middleware.UserContextKey).(*auth.CustomClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Pendelegasian eksekusi potong saldo (misal statis 100) ke Service
	newBalance, err := h.service.DeductBalance(r.Context(), claims.Username, 100)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientFunds) {
			http.Error(w, "Bad Request: Saldo tidak mencukupi", http.StatusBadRequest)
			return
		}
		log.Printf("Deduct error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Saldo berhasil dipotong. Sisa: %d", newBalance),
	})
}
