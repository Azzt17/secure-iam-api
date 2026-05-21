package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"secure-iam-api/internal/auth"
	"secure-iam-api/internal/middleware"
)

// simulasi database in-memory
type WalletDB struct {
	mu      sync.RWMutex // mengamankan shared state dari race condition
	balance int
}

// entity user
type User struct {
	Username     string
	PasswordHash string // jgn menyimpan password mentah
	Role         string
}

// database user
var userDB = struct {
	mu    sync.RWMutex
	users map[string]User
}{users: make(map[string]User)}

// inisialisasi saldo awal sistem
var masterWallet = &WalletDB{balance: 1000}

func main() {
	// multiplexer
	mux := http.NewServeMux()

	// registrasi endpoint
	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/wallet/deduct", deductWalletHandler)

	// endpoint pengujian recovery middleware
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Database meledak karena memory leak!")
	})

	mux.HandleFunc("/register", registerHandler)

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

	// ubah handler dari mux menjadi secureHandler
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      secureHandler,
		ReadTimeout:  5 * time.Second,   // batas waktu membaca request client
		WriteTimeout: 10 * time.Second,  // batas waktu server membalas
		IdleTimeout:  120 * time.Second, // batas waktu koneksi tetap hidup
	}

	log.Println("Secure IAM API berjalan di port :8080 dengan Middleware...")

	// mulai menerima request,
	// fungsi ini akan memblokir jalannya program sampai server di matikan
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server mati secara tidak wajar: %v", err)
	}
}

// -------- Handlers ------------
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
	Username string `json:"username"`
	Password string `json:"password"`
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
	if req.Username == "" || len(req.Password) < 6 {
		http.Error(w, "Bad Request: Username kosong atau password kurang dari 4 karakter", http.StatusBadRequest)
		return
	}

	// cek apakah user sudah ada (gunakan RLOCK utk read)
	userDB.mu.RLock()
	_, exists := userDB.users[req.Username]
	userDB.mu.RUnlock()

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

	// simpan ke database (gunakan lock utk write)
	userDB.mu.Lock()
	userDB.users[req.Username] = User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		Role:         "user", // default role
	}
	userDB.mu.Unlock()

	// respons
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "User berhasil didaftarkan",
	})
}
