package route

import (
	"github.com/xenvoid404/neko-tunneling/controller"
	"github.com/xenvoid404/neko-tunneling/middleware"
	"github.com/xenvoid404/neko-tunneling/pkg/validator"
	"github.com/xenvoid404/neko-tunneling/repository"
	"github.com/xenvoid404/neko-tunneling/service"
)

func (r *Router) SetupAPIRoutes() {
	validator := validator.NewValidator()
	userRepo := repository.NewUserRepository(r.DB.DB)

	sshService := service.NewSSHService()
	vmessService := service.NewVmessService(r.Cfg)
	vlessService := service.NewVlessService(r.Cfg)
	trojanService := service.NewTrojanService(r.Cfg)

	sshController := controller.NewSSHController(r.Cfg, validator, sshService, userRepo)
	vmessController := controller.NewVmessController(r.Cfg, validator, vmessService, userRepo)
	vlessController := controller.NewVlessController(r.Cfg, validator, vlessService, userRepo)
	trojanController := controller.NewTrojanController(r.Cfg, validator, trojanService, userRepo)

	authMiddleware := middleware.NewAuthMiddleware(r.Cfg)

	api := r.App.Group("/vps")
	api.Use(authMiddleware.Authenticate)

	api.Post("/trial/ssh", sshController.Trial)
	api.Post("/trial/vmess", vmessController.Trial)
	api.Post("/trial/vless", vlessController.Trial)
	api.Post("/trial/trojan", trojanController.Trial)

	api.Post("/create/ssh", sshController.Create)
	api.Post("/create/vmess", vmessController.Create)
	api.Post("/create/vless", vlessController.Create)
	api.Post("/create/trojan", trojanController.Create)

	api.Patch("/renew/ssh/:username", sshController.Renew)
	api.Patch("/renew/vmess/:username", vmessController.Renew)
	api.Patch("/renew/vless/:username", vlessController.Renew)
	api.Patch("/renew/trojan/:username", trojanController.Renew)

	api.Delete("/delete/ssh/:username", sshController.Delete)
	api.Delete("/delete/vmess/:username", vmessController.Delete)
	api.Delete("/delete/vless/:username", vlessController.Delete)
	api.Delete("/delete/trojan/:username", trojanController.Delete)
}
