package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"secure-iam-api/internal/middleware"
)

// simulasi database in-memory
type WalletDB struct {
	mu      sync.RWMutex // mengamankan shared state dari race condition
	balance int
}

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

	// memasang penghubung middleware
	// urutan: recover -> requestID -> logger -> mux
	secureHandler := middleware.Chain(
		mux,
		middleware.Recover,
		middleware.RequestID,
		middleware.Logger,
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
