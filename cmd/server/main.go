package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/droid-keyusage-go/internal/api"
	"github.com/droid-keyusage-go/internal/config"
	"github.com/droid-keyusage-go/internal/services"
	"github.com/droid-keyusage-go/internal/storage"
	"github.com/droid-keyusage-go/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	// Initialize logger
	log := utils.NewLogger()
	defer log.Sync()

	// Load configuration
	cfg := config.Load()
	log.Info("Configuration loaded",
		"redis_url", cfg.RedisURL,
		"max_workers", cfg.MaxWorkers,
		"port", cfg.Port,
	)

	// Initialize Redis
	redisClient, err := storage.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to Redis", "error", err)
	}
	defer redisClient.Close()

	log.Info("Connected to Redis successfully")

	// Initialize storage
	store := storage.NewStorage(redisClient)

	// Initialize services
	authService := services.NewAuthService(store, cfg.AdminPassword)
	workerPool := services.NewWorkerPool(cfg.MaxWorkers, cfg.QueueSize)
	apiKeyService := services.NewAPIKeyService(store, workerPool)

	// Start worker pool
	workerPool.Start()
	defer workerPool.Stop()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: api.ErrorHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		ServerHeader: "Droid-KeyUsage",
		AppName:      "Droid API Key Usage Monitor",
	})

	// Middlewares
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Asia/Shanghai",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Initialize handlers
	handlers := api.NewHandlers(apiKeyService, authService, cfg)

	// Setup routes
	api.SetupRoutes(app, handlers)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Info("Shutting down server...")
		
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			log.Error("Server shutdown error", "error", err)
		}
	}()

	// Start server
	log.Info("Starting server", "port", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server", "error", err)
	}
}
