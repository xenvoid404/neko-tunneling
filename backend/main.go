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

	"github.com/xenvoid404/neko-tunneling/api/route"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/database"
	"github.com/xenvoid404/neko-tunneling/docs"
	"github.com/xenvoid404/neko-tunneling/mux"
	"github.com/xenvoid404/neko-tunneling/pkg/logger"
	"github.com/xenvoid404/neko-tunneling/pkg/provision"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
)

// @Version                    1.0
// @BasePath                   /
// @Title                      Neko Tunneling API
// @Description                Selamat datang di dokumentasi Neko Tunneling API
// @SecurityDefinitions.apikey BearerAuth
// @In                         header
// @Name                       Authorization
// @Description                Masukkan token dengan format: Bearer {token}
func main() {
	cfg := config.GetConfig()
	logger.CreateLogger(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbCtx, cancelDB := context.WithTimeout(ctx, 10*time.Second)
	defer cancelDB()

	if err := database.Connect(dbCtx, cfg); err != nil {
		slog.Error("Gagal inisiasi database, aplikasi terhenti",
			slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()

	if err := provision.InitXrayClient(cfg.XrayAPIAddr); err != nil {
		slog.Error("Gagal inisiasi klien Xray API, aplikasi terhenti",
			slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		slog.Info("Menutup koneksi Xray API")
		if err := provision.CloseXrayClient(); err != nil {
			slog.Error("Gagal menutup koneksi Xray API",
				slog.Any("error", err))
		}
	}()

	docs.SwaggerInfo.Host = resolveSwaggerHost(cfg)

	var wg sync.WaitGroup
	wg.Add(2)
	go runMuxService(ctx, cfg, &wg, stop)
	go runAPIService(ctx, cfg, &wg, stop)

	wg.Wait()
	slog.Info("Semua service berhenti, aplikasi keluar")
}

func resolveSwaggerHost(cfg *config.Config) string {
	if domain := utils.ReadFile(cfg.CacheDomainPath); domain != "" {
		slog.Info("Swagger Host dikonfigurasi menggunakan Domain",
			slog.String("domain", domain))
		return domain
	}
	if ip := utils.ReadFile(cfg.CacheIPPath); ip != "" {
		slog.Info("Swagger Host dikonfigurasi menggunakan IP",
			slog.String("ip", ip))
		return ip
	}

	slog.Warn("File domain/IP tidak ditemukan, menggunakan fallback",
		slog.String("fallback", cfg.AppAddr))
	return cfg.AppAddr
}

func runMuxService(ctx context.Context, cfg *config.Config, wg *sync.WaitGroup, stop context.CancelFunc) {
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

func runAPIService(ctx context.Context, cfg *config.Config, wg *sync.WaitGroup, stop context.CancelFunc) {
	defer wg.Done()
	slog.Info("Memulai service Fiber...")

	app := fiber.New(fiber.Config{AppName: cfg.AppName})
	app.Use(recover.New())
	route.Setup(app, cfg)

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
