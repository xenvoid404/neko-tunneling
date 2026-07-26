package logger

import (
	"log/slog"
	"os"
)

func CreateLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
