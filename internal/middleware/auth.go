package middleware

import (
	"context"
	"net/http"

	"secure-iam-api/internal/auth"
)

const UserContextKey contextKey = "user_claims"

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ekstrak tiket dari Secure Cookie
		cookie, err := r.Cookie("access_token")
		if err != nil {
			http.Error(w, "Unauthorized: Silahkan login terlebih dahulu", http.StatusUnauthorized)
			return
		}

		// validasi keaslian tiket
		claims, err := auth.ValidateJWT(cookie.Value)
		if err != nil {
			// force logout jika token di manipulasi/expired
			http.SetCookie(w, &http.Cookie{
				Name:     "access_token",
				Value:    "",
				MaxAge:   -1,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			})
			http.Error(w, "Unauthorized: Sesi tidak valid atau telah berakhir", http.StatusUnauthorized)
			return
		}
		// simpan identitas user ke dalam context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
