package logger

import (
	"log/slog"
	"os"
	"strings"
)

// secureInterceptor mencegat setiap data (key-value) sebelum dicetak ke log
func secureInterceptor(groups []string, a slog.Attr) slog.Attr {
	// 1. blokir
	sensitiveKeys := map[string]bool{
		"password": true,
		"token":    true,
		"secret":   true,
		"pin":      true,
	}
	if sensitiveKeys[strings.ToLower(a.Key)] {
		a.Value = slog.StringValue("[REDACTED]")
		return a
	}

	// 2. masking utk PII
	if strings.ToLower(a.Key) == "email" {
		email := a.Value.String()
		a.Value = slog.StringValue(maskEmail(email))
		return a
	}

	return a
}

// maskEmail menyamarkan email (contoh: faridwajdi@example.com -> f***i@example.com)
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "[INVALID EMAIL_FORMAT]"
	}

	id := parts[0]
	domain := parts[1]

	if len(id) <= 2 {
		id = "***"
	} else {
		id = string(id[0]) + "***" + string(id[len(id)-1])
	}

	return id + "@" + domain
}

// menyiapkan global logger berformat JSON dengan interceptor keamanan
func Init() {
	opts := &slog.HandlerOptions{
		ReplaceAttr: secureInterceptor,
	}

	// JSONHandler untuk mencegah serangan Log Injection (\n)
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// jadikan default logger
	slog.SetDefault(logger)
}
