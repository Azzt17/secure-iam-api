package repository

import (
	"context"

	"secure-iam-api/internal/domain"
)

type IAMRepository interface {
	CheckUserExist(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, username, passwordHash, role string) (int, error)
	CreateWallet(ctx context.Context, userID int, initialBalance int) error
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	DeductWalletBalance(ctx context.Context, userID int, deductAmount int) (int, error)
}
