package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/database"
	"github.com/xenvoid404/neko-tunneling/docs"
	"github.com/xenvoid404/neko-tunneling/pkg/logger"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
)

var log = logger.CreateLogger()
var cfg = config.GetConfig()

// @Version     1.0
// @BasePath    /
// @Title       Neko Tunneling API
// @Description Selamat datang di dokumentasi Neko Tunneling API
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbCtx, cancel := context.WithTimeout(ctx, cfg.DBTimeout)
	defer cancel()

	if err := database.Connect(dbCtx); err != nil {
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

	docs.SwaggerInfo.Host = resolveSwaggerHost()

	log.Info("Memulai layanan Fiber...")

	app := fiber.New(fiber.Config{AppName: "Neko Tunneling API"})
	app.Get("/vps/docs/*", swaggo.HandlerDefault)

	go func() {
		if err := app.Listen(cfg.ListenAddr); err != nil {
			log.Error("Fiber terhentu karena error",
				slog.Any("error", err))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("Menerima sinyal shutdown, mematikan Fiber...")
}

func resolveSwaggerHost() string {
	if domain := utils.ReadFile("/var/lib/nekotun/cache/domain"); domain != "" {
		log.Info("Swagger Host dikonfigurasi menggunakan Domain",
			slog.String("domain", domain))
		return domain
	}
	if ip := utils.ReadFile("/var/lib/nekotun/cache/ip"); ip != "" {
		log.Info("Swagger Host dikonfigurasi menggunakan IP",
			slog.String("ip", ip))
		return ip
	}

	log.Warn("File domain/IP tidak ditemukan, menggunakan fallback",
		slog.String("fallback", cfg.ListenAddr))
	return cfg.ListenAddr
}
