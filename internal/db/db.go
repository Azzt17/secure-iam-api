package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// variabel global utk menampung connection pool
var Conn *sql.DB

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		slog.Info("WARNING: File .env tidak ditemukan, menggunakan environment variabel OS")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	Conn, err = sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("Kesalahan inisialisasi database", "error", err.Error())
	}

	// Connection Pool Architecture
	// batas maks koneksi simultan -> mencegah DB kehabisan RAM
	Conn.SetMaxOpenConns(25)

	// batas koneksi idle -> mengurangi beban CPU ketika traffic sepi
	Conn.SetMaxIdleConns(25)

	// waktu maksimal koneksi aktif sebelum di ganti dgn yg baru -> mencegah memory leak
	Conn.SetConnMaxLifetime(15 * time.Minute)

	if err = Conn.Ping(); err != nil {
		slog.Error("Gagal menghubungi PostgresSQL", "error", err.Error())
	}

	slog.Info("Koneksi PostgresSQL berhasil diamankan!")
}
