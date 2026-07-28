package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/model"
)

type AuthMiddleware struct {
	Cfg *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		Cfg: cfg,
	}
}

func (m *AuthMiddleware) Authenticate(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(&model.ErrorResponse{
			Success: false,
			Message: "api key tidak valid",
		})
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(&model.ErrorResponse{
			Success: false,
			Message: "api key tidak valid",
		})
	}

	token := parts[1]
	if token != m.Cfg.AppKey {
		return c.Status(fiber.StatusUnauthorized).JSON(&model.ErrorResponse{
			Success: false,
			Message: "api key tidak valid",
		})
	}

	return c.Next()
}
