package utils

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func AddSSHUser(username, password string, expiredAt time.Time) error {
	cmdAdd := exec.Command("useradd", "-M", "-N", "-s", "/bin/false", "-e", expiredAt.Format("2006-01-02"), username)
	if output, err := cmdAdd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd gagal: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	cmdPass := exec.Command("chpasswd")
	cmdPass.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", username, password))
	if output, err := cmdPass.CombinedOutput(); err != nil {
		_ = DeleteSSHUser(username)
		return fmt.Errorf("chpasswd gagal: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func DeleteSSHUser(username string) error {
	if output, err := exec.Command("userdel", username).CombinedOutput(); err != nil {
		return fmt.Errorf("userdel gagal: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}
