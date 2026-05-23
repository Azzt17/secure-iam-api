package domain

import "errors"

var (
	ErrUserNotFound      = errors.New("pengguna tidak ditemukan")
	ErrUserAlreadyExists = errors.New("username sudah terdaftar")
	ErrInsufficientFunds = errors.New("saldo tidak mencukupi")
	ErrWalletNotFound    = errors.New("dompet tidak ditemukan")
)
