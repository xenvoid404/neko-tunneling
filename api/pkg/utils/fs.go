package utils

import (
	"log/slog"
	"os"
	"strings"
)

func ReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Debug("Gagal membaca file cache",
			slog.String("path", path),
			slog.Any("error", err))
		return ""
	}
	return strings.TrimSpace(string(data))
}
