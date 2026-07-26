package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/pkg/provision"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
	"github.com/xenvoid404/neko-tunneling/repository"
)

type TrialAccount struct {
	Username  string
	Password  string
	ExpiredAt time.Time
}

func CreateTrial(ctx context.Context, cfg *config.Config, protocol string, expiredMinutes int) (*TrialAccount, error) {
	username := utils.RandomUsername()
	password := utils.RandomPassword(protocol)
	expiredAt := time.Now().Add(time.Duration(expiredMinutes) * time.Minute)

	execCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if protocol == "ssh" {
		if err := provision.AddSSHUser(execCtx, username, password, expiredAt); err != nil {
			return nil, fmt.Errorf("provisioning SSH gagal: %w", err)
		}
	} else {
		if err := provision.AddXrayUser(execCtx, protocol, password, username); err != nil {
			return nil, fmt.Errorf("provisioning Xray gagal: %w", err)
		}
	}

	user := repository.User{
		Protocol:   protocol,
		Username:   username,
		Password:   password,
		LimitIP:    0,
		LimitQuota: 0,
		Status:     "active",
		ExpiredAt:  expiredAt,
	}

	if err := repository.CreateUser(ctx, user); err != nil {
		rollbackProvisioning(execCtx, protocol, username)
		return nil, fmt.Errorf("gagal menyimpan akun trial: %w", err)
	}

	return &TrialAccount{Username: username, Password: password, ExpiredAt: expiredAt}, nil
}

func rollbackProvisioning(ctx context.Context, protocol, username string) {
	var err error
	if protocol == "ssh" {
		err = provision.DeleteSSHUser(ctx, username)
	} else {
		err = provision.RemoveXrayUser(ctx, protocol, username)
	}

	if err != nil {
		slog.Error("Rollback provisioning gagal, perlu pembersihan manual di sistem",
			slog.String("protocol", protocol),
			slog.String("username", username),
			slog.Any("error", err))
	}
}
