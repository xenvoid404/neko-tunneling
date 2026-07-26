package repository

import (
	"context"
	"fmt"

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
		return fmt.Errorf("gagal menyimpan usee: %w", err)
	}

	return nil
}
