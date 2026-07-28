package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/database"
	"github.com/xenvoid404/neko-tunneling/mux"
	"github.com/xenvoid404/neko-tunneling/pkg/logger"
	"github.com/xenvoid404/neko-tunneling/route"
)

func main() {
	cfg := config.GetConfig()
	logger.Setup(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db := database.Setup(ctx, cfg)
	defer db.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go runMux(ctx, cfg, &wg, stop)
	go runFiber(ctx, cfg, db, &wg, stop)
	wg.Wait()
	slog.Info("Semua service berhenti, aplikasi keluar")
}

func runMux(ctx context.Context, cfg *config.Config, wg *sync.WaitGroup, stop context.CancelFunc) {
	defer wg.Done()
	slog.Info("Memulai service Mux...")

	muxServer, err := mux.NewServer(cfg)
	if err != nil {
		slog.Error("Gagal menginisialisasi Mux",
			slog.Any("error", err))
		stop()
		return
	}

	if err := muxServer.Start(ctx); err != nil && ctx.Err() == nil {
		slog.Error("Mux berhenti karena error",
			slog.Any("error", err))
		stop()
		return
	}

	slog.Info("Service Mux berhenti")
}

func runFiber(ctx context.Context, cfg *config.Config, db *database.Database, wg *sync.WaitGroup, stop context.CancelFunc) {
	defer wg.Done()
	slog.Info("Memulai service Fiber...")

	app := fiber.New(fiber.Config{AppName: cfg.AppName})
	app.Use(recover.New())

	r := route.NewRouter(app, cfg, db)
	r.RegisterRoutes()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(cfg.AppAddr)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("Fiber terhenti karena error", slog.Any("error", err))
			stop()
		}
		return
	case <-ctx.Done():
		slog.Info("Menerima sinyal shutdown, mematikan Fiber...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		slog.Error("Gagal graceful shutdown Fiber", slog.Any("error", err))
	}
}
