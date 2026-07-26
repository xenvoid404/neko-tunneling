package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/xenvoid404/neko-tunneling/database"
)

type User struct {
	ID         int64
	Protocol   string
	Username   string
	Password   string
	LimitIP    int
	LimitQuota int
	Status     string
	ExpiredAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func CreateUser(ctx context.Context, data User) error {
	query := `
	INSERT INTO users (protocol, username, password, limit_ip, limit_quota, status, expired_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	if _, err := database.DB.ExecContext(ctx, query, data.Protocol, data.Username, data.Password, data.LimitIP, data.LimitQuota, data.Status, data.ExpiredAt); err != nil {
		return fmt.Errorf("gagal menyimpan user: %w", err)
	}

	return nil
}

func DeleteUserByUsername(ctx context.Context, username string) error {
	query := `DELETE FROM users WHERE username = ?`

	if _, err := database.DB.ExecContext(ctx, query, username); err != nil {
		return fmt.Errorf("gagal menghapus user: %w", err)
	}

	return nil
}

func ListExpired(ctx context.Context, before time.Time) ([]User, error) {
	query := `
	SELECT id, protocol, username, password, limit_ip, limit_quota, status, expired_at, created_at, updated_at
	FROM users WHERE status = 'active' AND expired_at <= ?
	`

	rows, err := database.DB.QueryContext(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil user expired: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Protocol, &u.Username, &u.Password, &u.LimitIP, &u.LimitQuota, &u.Status, &u.ExpiredAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("gagal membaca baris user expired: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal iterasi user expired: %w", err)
	}

	return users, nil
}

func MarkExpired(ctx context.Context, id int64) error {
	query := `UPDATE users SET status = 'expired' WHERE id = ?`
	if _, err := database.DB.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("gagal menandai user expired: %w", err)
	}
	return nil
}
