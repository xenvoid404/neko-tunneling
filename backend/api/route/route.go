package route

import (
	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"

	"github.com/xenvoid404/neko-tunneling/api/controller"
	"github.com/xenvoid404/neko-tunneling/api/middleware"
	"github.com/xenvoid404/neko-tunneling/config"
)

func Setup(app *fiber.App, cfg *config.Config) {
	app.Get("/vps/docs/*", swaggo.HandlerDefault)
	app.Use(middleware.AuthMiddleware(cfg.AppKey))
	app.Post("/vps/trial/ssh", controller.TrialSSH(cfg))
	app.Post("/vps/trial/vmess", controller.TrialVmess(cfg))
	app.Post("/vps/trial/vless", controller.TrialVless(cfg))
	app.Post("/vps/trial/trojan", controller.TrialTrojan(cfg))
}
