package middleware

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// tipe data custom untuk context key agar menghindari collison
type contextKey string

const RequestIDkey contextKey = "request_id"

// fungsi penghubung middleware
// ini memudahkan dalam penggunaan bnyk middleware tanpa membuat kode berantakan
type Middleware func(http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// Middleware 1: Panic Recovery
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// defer di ekseskusi tepat sebelum fungsi ini selesai atau ketika panic terjadi
		defer func() {
			if err := recover(); err != nil {
				// melakukan recover agar server tdk mati dan mencatat log internal
				// bukan di kembalikan ke stack trace user
				log.Printf("[PANIC RECOVERED: %v]", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r) // lanjut ke layer selanjutnya
	})
}

// Middleware 2: Request ID (Traceability)
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// simulasi pembuatan UUID, di prod gunakan library google/uuid
		reqID := "REQ-" + time.Now().Format("150405.000")

		// memasukkan ID kedalam context
		ctx := context.WithValue(r.Context(), RequestIDkey, reqID)

		// mengembalikan header ke client
		w.Header().Set("X-Request-ID", reqID)

		// meneruskan request dengan context yg sudah di modifikasi
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware 3: Logging
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// mengekstrak req id dari context
		reqID := r.Context().Value(RequestIDkey)
		if reqID == nil {
			reqID = "unknown"
		}

		// kirim ke handler utama
		next.ServeHTTP(w, r)

		// catat setelah handler utama selesai
		durasi := time.Since(start)
		log.Printf("[LOG] ID: %s | %s %s | Durasi: %v", reqID, r.Method, r.URL.Path, durasi)
	})
}

// Middleware 4: Security Headers (Browser Protection)
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// mencegah MIME Sniffing -> memaksa browser mengikuti context-type dari server
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// mencegah clickjacking -> tidak boleh di masukkan kedalam iframe oleh situs lain
		w.Header().Set("X-Frame-Options", "DENY")
		// memaksa HTTPS selama 1 tahun
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// membatasi resource apa saja yg boleh di muat browser -> mitigasi XSS
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

// Middleware 5: CORS (Cross-Origin Resource Sharing)
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hanya izinkan domain yg terdaftar di environtment
		allowedOrigin := os.Getenv("FRONTEND_URL")
		if allowedOrigin == "" {
			allowedOrigin = "https://localhost:3000" // safe fallback
		}
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// jika browser melakukan preflight request/options Method
		// langsung balas ok (200) tanpa mengeksekusi handler utama
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware 6: Rate Limiting (Anti Bruteforce & DoS)
// ini dibuat in-memory menggunakan sync.Mutex
type RateLimiter struct {
	visitor map[string]int // menyimpan jumlah request per ip
	mu      sync.Mutex
}

// Global instance utk rate limter (allow 5 request per ip)
var limiter = &RateLimiter{
	visitor: make(map[string]int),
}

// simulasi reset limiter per 10 detik (di prod bisa pke Redis)
func init() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			limiter.mu.Lock()
			limiter.visitor = make(map[string]int) // reset map
			limiter.mu.Unlock()
		}
	}()
}

func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// mengambil ip, di prod cek juga X-Forwarded-For kalau pakai Nginx/Cloudflare
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		limiter.mu.Lock()
		limiter.visitor[ip]++
		count := limiter.visitor[ip]
		limiter.mu.Unlock()

		// jika req lebih dari 5 dalam 10 detik, di tolak
		if count > 5 {
			http.Error(w, "Rate limit exceeded. Coba lagi nanti.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
