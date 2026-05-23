package service

import (
	"context"
	"errors"
	"testing"

	"secure-iam-api/internal/domain"
)

func TestDeductBalance(t *testing.T) {
	ctx := context.Background()

	// SKENARIO 1: Saldo mencukupi dan pemotongan berhasil
	t.Run("Success Path - Saldo Terpotong", func(t *testing.T) {
		// 1. Siapkan Database Mock dengan skenario sukses
		mockRepo := &MockIAMRepository{
			GetUserByUsernameFunc: func(ctx context.Context, username string) (*domain.User, error) {
				// user ditemukan di DB dengan ID 1
				return &domain.User{ID: 1, Username: "arsitek"}, nil
			},
			DeductWalletBalanceFunc: func(ctx context.Context, userID int, deductAmount int) (int, error) {
				// saldo awal 1000, dipotong 100, sisa 900
				return 900, nil
			},
		}

		// 2. Inject Mock ke dalam Service Layer
		svc := NewIAMService(mockRepo)

		// 3. Eksekusi fungsi bisnisnya
		newBalance, err := svc.DeductBalance(ctx, "arsitek", 100)
		// 4. Validasi (Assertion)
		if err != nil {
			t.Fatalf("Ekspektasi tidak ada error, tapi mendapat: %v", err)
		}
		if newBalance != 900 {
			t.Errorf("Ekspektasi sisa saldo 900, tapi mendapat: %d", newBalance)
		}
	})

	// SKENARIO 2: Saldo kurang dari jumlah yang diminta
	t.Run("Negative Path - Saldo Tidak Cukup", func(t *testing.T) {
		mockRepo := &MockIAMRepository{
			GetUserByUsernameFunc: func(ctx context.Context, username string) (*domain.User, error) {
				return &domain.User{ID: 1, Username: "arsitek"}, nil
			},
			DeductWalletBalanceFunc: func(ctx context.Context, userID int, deductAmount int) (int, error) {
				// DB menolak karena saldo kurang
				return 0, domain.ErrInsufficientFunds
			},
		}

		svc := NewIAMService(mockRepo)

		// Coba potong saldo 5000 (jauh di atas batas)
		_, err := svc.DeductBalance(ctx, "arsitek", 5000)

		// Validasi bahwa error yang ditangkap Service adalah tipe ErrInsufficientFunds
		if !errors.Is(err, domain.ErrInsufficientFunds) {
			t.Errorf("Ekspektasi error ErrInsufficientFunds, tapi mendapat: %v", err)
		}
	})
}
