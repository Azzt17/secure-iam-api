package middleware

import (
	"context"
	"log"
	"net/http"
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
