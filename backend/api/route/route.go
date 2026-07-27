package route

import (
	"html/template"

	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"

	"github.com/xenvoid404/neko-tunneling/api/controller"
	"github.com/xenvoid404/neko-tunneling/api/middleware"
	"github.com/xenvoid404/neko-tunneling/config"
)

func Setup(app *fiber.App, cfg *config.Config) {
	app.Get("/vps/docs/*", swaggo.New(swaggo.Config{
		Title: "Neko Tunneling API Docs",
		CustomStyle: `
			body { margin: 0; padding: 0; }
			.swagger-ui .wrapper { padding: 0 10px; }
			.swagger-ui .table-container { overflow-x: auto; }
			.swagger-ui .parameters-col_description { min-width: 150px; }
		`,
		CustomScript: `
			document.addEventListener("DOMContentLoaded", function() {
				var meta = document.createElement('meta');
				meta.name = 'viewport';
				meta.content = 'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no';
				document.getElementsByTagName('head')[0].appendChild(meta);
			});
		`,
		Plugins: []template.JS{
			template.JS(`{
				statePlugins: {
					auth: {
						wrapActions: {
							authorize: (ori) => (auth) => {
								Object.keys(auth).forEach(function (key) {
									var entry = auth[key];
									if (entry && typeof entry.value === "string") {
										var trimmed = entry.value.trim();
										if (trimmed && !/^bearer\s/i.test(trimmed)) {
											entry.value = "Bearer " + trimmed;
										}
									}
								});
								return ori(auth);
							}
						}
					}
				}
			}`),
		},
		DocExpansion: "list",
		DeepLinking:  true,
	}))
	app.Use(middleware.AuthMiddleware(cfg.AppKey))
	app.Post("/vps/trial/ssh", controller.TrialSSH(cfg))
	app.Post("/vps/trial/vmess", controller.TrialVmess(cfg))
	app.Post("/vps/trial/vless", controller.TrialVless(cfg))
	app.Post("/vps/trial/trojan", controller.TrialTrojan(cfg))
}
