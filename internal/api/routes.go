package api

import (
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all routes
func SetupRoutes(app *fiber.App, handlers *Handlers) {
	// Health check
	app.Get("/health", handlers.Health)

	// Authentication routes (no auth middleware)
	app.Post("/api/login", handlers.Login)
	app.Post("/api/logout", handlers.Logout)

	// API routes group with auth middleware
	api := app.Group("/api", AuthMiddleware(handlers.authService))
	
	// Data endpoints
	api.Get("/data", handlers.GetData)
	
	// API Key management
	api.Get("/keys", handlers.GetKeys)
	api.Post("/keys", handlers.AddKey)
	api.Post("/keys/import", handlers.ImportKeys)
	api.Get("/keys/:id/full", handlers.GetFullKey)
	api.Delete("/keys/:id", handlers.DeleteKey)
	api.Post("/keys/batch-delete", handlers.BatchDeleteKeys)

	// Serve static files
	app.Static("/", "./web/static", fiber.Static{
		Browse: false,
		Index:  "index.html",
	})
}
