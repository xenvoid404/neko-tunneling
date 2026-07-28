package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/xenvoid404/neko-tunneling/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, data model.User) error {
	query := `
	INSERT INTO users (protocol, username, password, limit_ip, limit_quota, status, expired_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	if _, err := r.db.ExecContext(ctx, query, data.Protocol, data.Username, data.Password, data.LimitIP, data.LimitQuota, data.Status, data.ExpiredAt); err != nil {
		return fmt.Errorf("gagal menyimpan user: %w", err)
	}

	return nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
	SELECT id, protocol, username, password, limit_ip, limit_quota, status, expired_at, created_at, updated_at
	FROM users WHERE username = ?
	`

	var u model.User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Protocol, &u.Username, &u.Password, &u.LimitIP, &u.LimitQuota, &u.Status, &u.ExpiredAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari user berdasarkan username: %w", err)
	}

	return &u, nil
}

func (r *UserRepository) UpdateExpiredByUsername(ctx context.Context, username string, expired time.Time) error {
	query := `
	UPDATE users 
	SET expired_at = ? 
	WHERE username = ?
	`

	result, err := r.db.ExecContext(ctx, query, expired, username)
	if err != nil {
		return fmt.Errorf("gagal update masa aktif user %s: %w", username, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal mengecek status update: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user tidak ditemukan")
	}

	return nil
}

func (r *UserRepository) DeleteByUsername(ctx context.Context, username string) error {
	query := `DELETE FROM users WHERE username = ?`

	result, err := r.db.ExecContext(ctx, query, username)
	if err != nil {
		return fmt.Errorf("gagal menghapus user %s dari database: %w", username, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal mengecek status delete: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user tidak ditemukan atau sudah terhapus")
	}

	return nil
}
