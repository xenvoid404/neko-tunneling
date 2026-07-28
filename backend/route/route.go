package route

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/database"
)

type Router struct {
	App *fiber.App
	Cfg *config.Config
	DB  *database.Database
}

func NewRouter(app *fiber.App, cfg *config.Config, db *database.Database) *Router {
	return &Router{
		App: app,
		Cfg: cfg,
		DB:  db,
	}
}

func (r *Router) RegisterRoutes() {
	r.SetupWebRoutes()
	r.SetupAPIRoutes()
}
