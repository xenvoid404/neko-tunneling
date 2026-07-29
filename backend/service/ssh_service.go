package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type SSHService struct{}

func NewSSHService() *SSHService {
	return &SSHService{}
}

func (s *SSHService) AddUser(ctx context.Context, username, password string) error {
	cmdAdd := exec.CommandContext(ctx, "useradd", "-M", "-N", "-s", "/bin/false", username)
	if output, err := cmdAdd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd gagal: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	cmdPass := exec.CommandContext(ctx, "chpasswd")
	cmdPass.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", username, password))
	if output, err := cmdPass.CombinedOutput(); err != nil {
		_ = s.DelUser(ctx, username)
		return fmt.Errorf("chpasswd gagal: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *SSHService) DelUser(ctx context.Context, username string) error {
	if output, err := exec.CommandContext(ctx, "userdel", username).CombinedOutput(); err != nil {
		return fmt.Errorf("userdel gagal: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}
