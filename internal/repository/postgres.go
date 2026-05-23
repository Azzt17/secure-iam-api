package repository

import (
	"context"
	"database/sql"
	"errors"

	"secure-iam-api/internal/domain"
)

type postgresRepo struct {
	db *sql.DB
}

// dependency injection constructor
func NewPostgresRepository(db *sql.DB) IAMRepository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) CheckUserExist(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)"
	err := r.db.QueryRowContext(ctx, query, username).Scan(&exists)
	return exists, err
}

func (r *postgresRepo) CreateUser(ctx context.Context, username, passwordHash, role string) (int, error) {
	var userID int
	query := "INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3) RETURNING id"
	err := r.db.QueryRowContext(ctx, query, username, passwordHash, role).Scan(&userID)
	return userID, err
}

func (r *postgresRepo) CreateWallet(ctx context.Context, userID int, initialBalance int) error {
	query := "INSERT INTO wallets (user_id, balance) VALUES ($1, $2)"
	_, err := r.db.ExecContext(ctx, query, userID, initialBalance)
	return err
}

func (r *postgresRepo) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	query := "SELECT id, username, password_hash, role FROM users WHERE username = $1"
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *postgresRepo) DeductWalletBalance(ctx context.Context, userID int, deductAmount int) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var currentBalance int
	// TOCTOU Mitigation & Row-Level Locking
	err = tx.QueryRowContext(ctx, "SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE", userID).Scan(&currentBalance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.ErrWalletNotFound
		}
		return 0, err
	}

	if currentBalance < deductAmount {
		return 0, domain.ErrInsufficientFunds // Rollback otomatis terjadi
	}

	newBalance := currentBalance - deductAmount
	_, err = tx.ExecContext(ctx, "UPDATE wallets SET balance = $1 WHERE user_id = $2", newBalance, userID)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return newBalance, nil
}
