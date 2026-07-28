package route

import (
	"html/template"
	"log/slog"

	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/xenvoid404/neko-tunneling/docs"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
)

// @Title                      Neko Tunneling API
// @Version                    1.0
// @Description                Selamat datang di dokumentasi API Neko Tunneling. Halaman ini berisi referensi endpoint lengkap untuk mempermudah integrasi VPS dengan Web Panel, skrip automasi, maupun Bot Telegram.
// @BasePath                   /
// @SecurityDefinitions.apikey BearerAuth
// @In                         header
// @Name                       Authorization
// @Description                Masukkan token Anda saja (Awalan "Bearer " akan ditambahkan otomatis)
func (r *Router) SetupWebRoutes() {
	docs.SwaggerInfo.Host = r.resolveSwaggerHost()
	r.App.Get("/vps/docs/*", swaggo.New(swaggo.Config{
		Title:        "Neko Tunneling API Docs",
		DocExpansion: "list",
		DeepLinking:  true,
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
	}))
}

func (r *Router) resolveSwaggerHost() string {
	if domain := utils.ReadFile(r.Cfg.CacheDomainPath); domain != "" {
		slog.Info("Swagger Host dikonfigurasi menggunakan Domain",
			slog.String("domain", domain))
		return domain
	}
	if ip := utils.ReadFile(r.Cfg.CacheIPPath); ip != "" {
		slog.Info("Swagger Host dikonfigurasi menggunakan IP",
			slog.String("ip", ip))
		return ip
	}

	slog.Warn("File domain/IP tidak ditemukan, menggunakan fallback",
		slog.String("fallback", r.Cfg.AppAddr))
	return r.Cfg.AppAddr
}
