package service

import (
	"context"

	"secure-iam-api/internal/domain"
)

// MockIAMRepository adalah tiruan dari database untuk keperluan testing.
// Ini mematuhi kontrak repository.IAMRepository.
type MockIAMRepository struct {
	CheckUserExistFunc      func(ctx context.Context, username string) (bool, error)
	CreateUserFunc          func(ctx context.Context, username, passwordHash, role string) (int, error)
	CreateWalletFunc        func(ctx context.Context, userID int, initialBalance int) error
	GetUserByUsernameFunc   func(ctx context.Context, username string) (*domain.User, error)
	DeductWalletBalanceFunc func(ctx context.Context, userID int, deductAmount int) (int, error)
}

func (m *MockIAMRepository) CheckUserExist(ctx context.Context, username string) (bool, error) {
	return m.CheckUserExistFunc(ctx, username)
}

func (m *MockIAMRepository) CreateUser(ctx context.Context, username, passwordHash, role string) (int, error) {
	return m.CreateUserFunc(ctx, username, passwordHash, role)
}

func (m *MockIAMRepository) CreateWallet(ctx context.Context, userID int, initialBalance int) error {
	return m.CreateWalletFunc(ctx, userID, initialBalance)
}

func (m *MockIAMRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	return m.GetUserByUsernameFunc(ctx, username)
}

func (m *MockIAMRepository) DeductWalletBalance(ctx context.Context, userID int, deductAmount int) (int, error) {
	return m.DeductWalletBalanceFunc(ctx, userID, deductAmount)
}
