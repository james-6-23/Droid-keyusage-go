package api

import (
	"embed"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

// SetupRoutes configures all routes
func SetupRoutes(app *fiber.App, handlers *Handlers, staticFiles embed.FS) {
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
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(staticFiles),
		PathPrefix: "web/static",
		Browse:     false,
		Index:      "index.html",
	}))

	// Fallback to index.html for SPA
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/static/index.html")
	})
}
