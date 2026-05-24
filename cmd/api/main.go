package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"secure-iam-api/internal/db"
	handler "secure-iam-api/internal/handlers"
	"secure-iam-api/internal/middleware"
	"secure-iam-api/internal/repository"
	"secure-iam-api/internal/service"

	"github.com/go-playground/validator/v10"
)

func main() {
	// Inisialisasi Database
	db.InitDB()
	defer func() {
		if err := db.Conn.Close(); err != nil {
			log.Printf("Gagal menutup koneksi database: %v", err)
		}
	}()

	// validasi jwt secret key
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("FATAL: JWT_SECRET tidak dikonfigurasi di environtment")
	}

	// Inisialisasi Validator
	validate := validator.New()

	// root context utk kontrol Background task
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	rateLimiter := middleware.NewRateLimiter(rootCtx)

	// DEPENDENCY INJECTION
	// Suntikkan koneksi DB ke Repository
	repo := repository.NewPostgresRepository(db.Conn)
	// Suntikkan Repository ke Service
	svc := service.NewIAMService(repo)

	// Inisialisasi router dan handlers
	mux := http.NewServeMux()
	iamHandler := handler.NewIAMHandler(svc, validate)

	// Endpoint Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Rute Publik (Dilindungi Rate Limiter)
	mux.Handle("/register", rateLimiter(http.HandlerFunc(iamHandler.Register)))
	mux.Handle("/login", rateLimiter(http.HandlerFunc(iamHandler.Login)))

	// Rute Privat (Dilindungi Rate Limiter + JWT Gatekeeper)
	mux.Handle("/wallet/deduct", rateLimiter(middleware.RequireAuth(http.HandlerFunc(iamHandler.DeductWallet))))

	// Dekorasi Layer 7 Terakhir (Security Headers, CORS, Panic Recovery, Logger)
	var finalHandler http.Handler = mux
	finalHandler = middleware.SecurityHeaders(finalHandler)
	finalHandler = middleware.CORS(finalHandler)
	finalHandler = middleware.Recover(finalHandler)
	finalHandler = middleware.RequestID(finalHandler)
	finalHandler = middleware.Logger(finalHandler)

	// Konfigurasi Transport (TLS)
	tlsConfig := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	server := &http.Server{
		Addr:         ":8443",
		Handler:      finalHandler,
		TLSConfig:    tlsConfig,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Secure IAM API berjalan di jalur TERENKRIPSI port :8443...")
		if err := server.ListenAndServeTLS("certs/server.crt", "certs/server.key"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server gagal dijalankan: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Sinyal penghentian diterima. Mematikan server secara perlahan (Graceful Shutdown)...")

	// picu pembatalan root context
	rootCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server dipaksa mati karena timeout: %v", err)
	}

	log.Println("Server berhasil dihentikan dengan aman.")
}
