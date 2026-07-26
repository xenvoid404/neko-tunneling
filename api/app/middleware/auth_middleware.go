package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/xenvoid404/neko-tunneling/app/dto"
)

func AuthMiddleware(apiKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(&dto.ErrorRes{
				Success: false,
				Message: "api key tidak valid",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(&dto.ErrorRes{
				Success: false,
				Message: "api key tidak valid",
			})
		}

		token := parts[1]
		if token != apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(&dto.ErrorRes{
				Success: false,
				Message: "api key tidak valid",
			})
		}

		return c.Next()
	}
}
