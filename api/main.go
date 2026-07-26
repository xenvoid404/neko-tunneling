package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/xenvoid404/neko-tunneling/app/route"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/database"
	"github.com/xenvoid404/neko-tunneling/docs"
	"github.com/xenvoid404/neko-tunneling/pkg/logger"
	"github.com/xenvoid404/neko-tunneling/pkg/provision"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
)

var log = logger.CreateLogger()

// @Version     1.0
// @BasePath    /
// @Title       Neko Tunneling API
// @Description Selamat datang di dokumentasi Neko Tunneling API
func main() {
	cfg := config.GetConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbCtx, cancelDB := context.WithTimeout(ctx, 10*time.Second)
	defer cancelDB()

	if err := database.Connect(dbCtx, cfg); err != nil {
		log.Error("Gagal inisiasi database, aplikasi terhenti",
			slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()

	if err := provision.InitXrayClient(cfg.XrayAPIAddr); err != nil {
		log.Error("Gagal inisiasi klien Xray API, aplikasi terhenti",
			slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		log.Info("Menutup koneksi Xray API")
		if err := provision.CloseXrayClient(); err != nil {
			log.Error("Gagal menutup koneksi Xray API",
				slog.Any("error", err))
		}
	}()

	docs.SwaggerInfo.Host = resolveSwaggerHost(cfg)

	app := fiber.New(fiber.Config{AppName: cfg.AppName})
	app.Use(recover.New())
	route.Setup(app, cfg)

	log.Info("Memulai layanan Fiber...")

	go func() {
		if err := app.Listen(cfg.AppAddr); err != nil {
			log.Error("Fiber terhenti karena error",
				slog.Any("error", err))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("Menerima sinyal shutdown, mematikan Fiber...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("Gagal graceful shutdown Fiber",
			slog.Any("error", err))
	}
}

func resolveSwaggerHost(cfg *config.Config) string {
	if domain := utils.ReadFile(cfg.CacheDomainPath); domain != "" {
		log.Info("Swagger Host dikonfigurasi menggunakan Domain",
			slog.String("domain", domain))
		return domain
	}
	if ip := utils.ReadFile(cfg.CacheIPPath); ip != "" {
		log.Info("Swagger Host dikonfigurasi menggunakan IP",
			slog.String("ip", ip))
		return ip
	}

	log.Warn("File domain/IP tidak ditemukan, menggunakan fallback",
		slog.String("fallback", cfg.AppAddr))
	return cfg.AppAddr
}
