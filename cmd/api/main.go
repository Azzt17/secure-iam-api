package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"secure-iam-api/internal/auth"
	"secure-iam-api/internal/db"
	"secure-iam-api/internal/middleware"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// simulasi database in-memory
type WalletDB struct {
	mu      sync.RWMutex // mengamankan shared state dari race condition
	balance int
}

// inisialisasi saldo awal sistem
var masterWallet = &WalletDB{balance: 1000}

func main() {
	// nyalakan database
	db.InitDB()

	// multiplexer
	mux := http.NewServeMux()

	// endpoint publik (tidak butuh token)
	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/login", loginHandler)

	// endpoint private (RequireAuth)
	secureWalletHandler := middleware.RequireAuth(http.HandlerFunc(deductWalletHandler))
	mux.Handle("/wallet/deduct", secureWalletHandler)

	// endpoint pengujian recovery middleware
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Database meledak karena memory leak!")
	})

	// memasang penghubung middleware
	// urutan: Recover -> Logger -> Ratelimit -> CORS -> SecurityHeaders
	secureHandler := middleware.Chain(
		mux,
		middleware.Recover,
		middleware.RequestID,
		middleware.Logger,
		middleware.RateLimit,
		middleware.CORS,
		middleware.SecurityHeaders,
	)

	// konfigurasi keamanan transport layer
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12, // versi di bawah ini rentan
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
	}

	srv := &http.Server{
		Addr:         ":8443",
		Handler:      secureHandler,
		TLSConfig:    tlsConfig,
		ReadTimeout:  5 * time.Second,   // batas waktu membaca request client
		WriteTimeout: 10 * time.Second,  // batas waktu server membalas
		IdleTimeout:  120 * time.Second, // batas waktu koneksi tetap hidup
	}

	log.Println("Secure IAM API berjalan di port :8443...")

	// mulai menerima request,
	// fungsi ini akan memblokir jalannya program sampai server di matikan
	// *menggunakan sertifikat keamanan terenkripsi
	if err := srv.ListenAndServeTLS("certs/server.crt", "certs/server.key"); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server mati secara tidak wajar: %v", err)
	}
}

// -------- Handlers ------------

func formatValidationError(err error) map[string]string {
	errs := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		errs[err.Field()] = "Format tidak valid (Gagal pada aturan: " + err.Tag() + ")"
	}
	return errs
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// filter method: endpoint ini hanya menerima get
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// set header -> set status code -> write body :
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "OK", "message": "API berfungsi normal"}`))
}

func deductWalletHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// mencegah Time-of-Check to Time-of-Use (TOCTOU)
	// mengunci memori agar tidak ada goroutine lain yg bisa melakukan write saldo
	// di saat yg sama
	masterWallet.mu.Lock()
	defer masterWallet.mu.Unlock()

	if masterWallet.balance >= 100 {
		// simulasi delay pemrosesan database
		time.Sleep(10 * time.Millisecond)

		masterWallet.balance -= 100

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "succes",
			"message":    "Saldo berhasil dipotong 100",
			"sisa_saldo": masterWallet.balance,
		})
		return
	}

	// jika saldo kurang, kirim bad request
	http.Error(w, "Saldo tidak mencukupi", http.StatusBadRequest)
}

// pemetaan JSON req dari client
type RegisterRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=4,max=32"`
	Password string `json:"password" validate:"required,min=8"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// parsing payload JSON dari client
	var req RegisterRequest
	// limit body size
	r.Body = http.MaxBytesReader(w, r.Body, 1048576) // max 1 MB

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: Format JSON tidak valid", http.StatusBadRequest)
		return
	}

	// validasi dasar
	//	if req.Username == "" || len(req.Password) < 6 {
	//		http.Error(w, "Bad Request: Username kosong atau password kurang dari 4 karakter", http.StatusBadRequest)
	//		return
	//	}
	// Security validator
	if err := validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity) // http 422
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Validasi Input Gagal",
			"details": formatValidationError(err),
		})
		return
	}

	// cek ketersediaan user menggunakan parameterized query
	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)"
	err := db.Conn.QueryRow(checkQuery, req.Username).Scan(&exists)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "Conflict: Username sudah terdaftar", http.StatusConflict)
		return
	}

	// kriptografi
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// simpan user ke database dan ambil ID menggunakan RETURN
	var userID int
	insertUserQuery := "INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3) RETURNING id"
	err = db.Conn.QueryRow(insertUserQuery, req.Username, hashedPassword, "user").Scan(&userID)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// buat wallet otomatis untuk user baru agar memiliki relasi (foreign key)
	// saldo awal: 1000
	insertWalletQuery := "INSERT INTO wallets (user_id, balance) VALUES ($1, $2)"
	_, err = db.Conn.Exec(insertWalletQuery, userID, 1000)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// respons
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "User berhasil didaftarkan",
	})
}

type LoginRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=4,max=32"`
	Password string `json:"password" validate:"required"`
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// parsing payload JSON
	var req LoginRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Security Validator
	if err := validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity) // http 422
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Validasi Input Gagal",
			"details": formatValidationError(err),
		})
		return
	}

	// tempat menampung data dari baris database
	var dbUsername string
	var dbPasswordHash string
	var dbRole string

	// query data berdasarkan satu parameter ($1)
	loginQuery := "SELECT username, password_hash, role FROM users WHERE username = $1"
	err := db.Conn.QueryRow(loginQuery, req.Username).Scan(&dbUsername, &dbPasswordHash, &dbRole)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Kredensial tidak valid", http.StatusUnauthorized)
			return
		}
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// validasi password
	if !auth.CheckPasswordHash(req.Password, dbPasswordHash) {
		http.Error(w, "Kredensial tidak valid", http.StatusUnauthorized)
		return
	}

	// generate JWT
	tokenString, err := auth.GenerateJWT(dbUsername, dbRole)
	if err != nil {
		http.Error(w, "Gagal membuat sesi", http.StatusInternalServerError)
		return
	}

	// secure session managament
	// tokennya tdk di kirim via JSON body, melainkan via Cookie yg dikunci
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenString,
		Expires:  time.Now().Add(1 * time.Hour),
		HttpOnly: true, // tdk bisa di baca javascript -> mitigasi XSS
		Secure:   true, // sudah menggunakan HTTPS (tls)
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Login berhasil",
	})
}
