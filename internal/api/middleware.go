package api

import (
	"github.com/droid-keyusage-go/internal/models"
	"github.com/droid-keyusage-go/internal/services"
	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware checks if the user is authenticated
func AuthMiddleware(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip auth for health check and static files
		path := c.Path()
		if path == "/health" || path == "/api/login" {
			return c.Next()
		}

		// Check if auth is required
		if !authService.IsAuthRequired() {
			return c.Next()
		}

		// Check session cookie
		sessionID := c.Cookies("session")
		if sessionID != "" && authService.ValidateSession(sessionID) {
			return c.Next()
		}

		// Check Authorization header (for API calls)
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer <token>" format
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				if authService.ValidateJWT(token) {
					return c.Next()
				}
			}
		}

		// Return 401 for API requests
		if len(path) > 4 && path[:4] == "/api" {
			return c.Status(401).JSON(models.ErrorResponse{Error: "Unauthorized"})
		}

		// Redirect to login page for web requests
		return c.Redirect("/login")
	}
}

// ErrorHandler handles global errors
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// API error response
	if c.Path()[:4] == "/api" {
		return c.Status(code).JSON(models.ErrorResponse{
			Error: message,
		})
	}

	// HTML error response
	return c.Status(code).SendString(message)
}
