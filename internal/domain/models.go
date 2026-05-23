package domain

import "time"

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

type Wallet struct {
	ID       int
	UserID   int
	Balance  int
	UpdateAt time.Time
}
