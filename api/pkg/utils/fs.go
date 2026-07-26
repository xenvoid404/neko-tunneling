package utils

import (
	"log/slog"
	"os"
	"strings"

	"github.com/xenvoid404/neko-tunneling/pkg/logger"
)

var log = logger.CreateLogger()

func ReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Debug("Gagal membaca file cache",
			slog.String("path", path),
			slog.Any("error", err))
		return ""
	}
	return strings.TrimSpace(string(data))
}
