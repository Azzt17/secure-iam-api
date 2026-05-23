package service

import (
	"context"
	"errors"

	"secure-iam-api/internal/auth"
	"secure-iam-api/internal/domain"
	"secure-iam-api/internal/repository"
)

// IAMService adalah kontrak interface untuk logika bisnis aplikasi
type IAMService interface {
	RegisterUser(ctx context.Context, username, password string) error
	LoginUser(ctx context.Context, username, password string) (string, error)
	DeductBalance(ctx context.Context, username string, amount int) (int, error)
}

// iamService adalah implementasi nyata dari kontrak bisnis
type iamService struct {
	repo repository.IAMRepository
}

func NewIAMService(repo repository.IAMRepository) IAMService {
	return &iamService{repo: repo}
}

func (s *iamService) RegisterUser(ctx context.Context, username, password string) error {
	// 1. Cek ketersediaan username
	exists, err := s.repo.CheckUserExist(ctx, username)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrUserAlreadyExists
	}

	// 2. Hashing
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	// 3. Simpan User
	userID, err := s.repo.CreateUser(ctx, username, hashedPassword, "user")
	if err != nil {
		return err
	}

	// 4. Setiap user baru mendapat dompet berisi 1000
	err = s.repo.CreateWallet(ctx, userID, 1000)
	if err != nil {
		return err
	}

	return nil
}

func (s *iamService) LoginUser(ctx context.Context, username, password string) (string, error) {
	// 1. Ambil data asli dari repository
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", err
	}

	// 2. Validasi Hash
	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return "", errors.New("kredensial tidak valid")
	}

	// 3. Generate JWT
	tokenString, err := auth.GenerateJWT(user.Username, user.Role)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *iamService) DeductBalance(ctx context.Context, username string, amount int) (int, error) {
	// 1. Service Layer harus mengubah username (dari JWT) menjadi UserID untuk Database
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, err
	}

	// 2. Pendelegasian eksekusi pemotongan persisten ke Repository
	newBalance, err := s.repo.DeductWalletBalance(ctx, user.ID, amount)
	if err != nil {
		return 0, err
	}

	return newBalance, nil
}
