package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"

	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/controller"
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

	dbCtx, cancelDB := context.WithTimeout(ctx, cfg.DBTimeout)
	defer cancelDB()

	if err := database.Connect(dbCtx, cfg); err != nil {
		log.Error("Gagal inisiasi database, aplikasi terhenti",
			slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		log.Info("Menutup koneksi database")
		if err := database.DB.Close(); err != nil {
			log.Error("Gagal menutup koneksi database",
				slog.Any("error", err))
		}
	}()

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

	app := fiber.New(fiber.Config{AppName: "Neko Tunneling API"})
	app.Use(recover.New())
	app.Get("/vps/docs/*", swaggo.HandlerDefault)
	app.Post("/vps/trial/ssh", controller.TrialSSH(cfg))
	app.Post("/vps/trial/vmess", controller.TrialVmess(cfg))
	app.Post("/vps/trial/vless", controller.TrialVless(cfg))
	app.Post("/vps/trial/trojan", controller.TrialTrojan(cfg))

	log.Info("Memulai layanan Fiber...")

	go func() {
		if err := app.Listen(cfg.ListenAddr); err != nil {
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
	if domain := utils.ReadFile(cfg.CacheDir + "/domain"); domain != "" {
		log.Info("Swagger Host dikonfigurasi menggunakan Domain",
			slog.String("domain", domain))
		return domain
	}
	if ip := utils.ReadFile(cfg.CacheDir + "/ip"); ip != "" {
		log.Info("Swagger Host dikonfigurasi menggunakan IP",
			slog.String("ip", ip))
		return ip
	}

	log.Warn("File domain/IP tidak ditemukan, menggunakan fallback",
		slog.String("fallback", cfg.ListenAddr))
	return cfg.ListenAddr
}
