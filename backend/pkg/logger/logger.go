package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/xenvoid404/neko-tunneling/config"
)

func CreateLogger(cfg *config.Config) *slog.Logger {
	levelStr := strings.ToLower(cfg.LogLevel)
	formatStr := strings.ToLower(cfg.LogFormat)

	var logLevel slog.Level
	switch levelStr {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: logLevel}

	var handler slog.Handler
	if formatStr == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
