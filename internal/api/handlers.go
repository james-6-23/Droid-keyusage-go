package api

import (
	"time"

	"github.com/droid-keyusage-go/internal/config"
	"github.com/droid-keyusage-go/internal/models"
	"github.com/droid-keyusage-go/internal/services"
	"github.com/gofiber/fiber/v2"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	apiKeyService *services.APIKeyService
	authService   *services.AuthService
	config        *config.Config
}

// NewHandlers creates new handlers
func NewHandlers(apiKeyService *services.APIKeyService, authService *services.AuthService, cfg *config.Config) *Handlers {
	return &Handlers{
		apiKeyService: apiKeyService,
		authService:   authService,
		config:        cfg,
	}
}

// Health check endpoint
func (h *Handlers) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	})
}

// Login handles authentication
func (h *Handlers) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Invalid request"})
	}

	if !h.authService.ValidatePassword(req.Password) {
		return c.Status(401).JSON(models.ErrorResponse{Error: "Invalid password"})
	}

	// Create session
	sessionID, err := h.authService.CreateSession()
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: "Failed to create session"})
	}

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    sessionID,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	return c.JSON(models.SuccessResponse{Success: true})
}

// Logout handles logout
func (h *Handlers) Logout(c *fiber.Ctx) error {
	sessionID := c.Cookies("session")
	if sessionID != "" {
		_ = h.authService.DeleteSession(sessionID)
	}

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	return c.JSON(models.SuccessResponse{Success: true})
}

// GetData returns aggregated usage data
func (h *Handlers) GetData(c *fiber.Ctx) error {
	data, err := h.apiKeyService.GetAggregatedData()
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(data)
}

// GetKeys returns all API keys (masked)
func (h *Handlers) GetKeys(c *fiber.Ctx) error {
	keys, err := h.apiKeyService.GetAllKeys()
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(keys)
}

// GetFullKey returns the full API key
func (h *Handlers) GetFullKey(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Key ID required"})
	}

	key, err := h.apiKeyService.GetFullKey(id)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: err.Error()})
	}

	if key == nil {
		return c.Status(404).JSON(models.ErrorResponse{Error: "Key not found"})
	}

	return c.JSON(fiber.Map{
		"id":  key.ID,
		"key": key.Key,
	})
}

// ImportKeys handles batch import
func (h *Handlers) ImportKeys(c *fiber.Ctx) error {
	var req models.ImportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Invalid request"})
	}

	if len(req.Keys) == 0 {
		return c.Status(400).JSON(models.ErrorResponse{Error: "No keys provided"})
	}

	result, err := h.apiKeyService.ImportKeys(req.Keys)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(result)
}

// DeleteKey deletes a single API key
func (h *Handlers) DeleteKey(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Key ID required"})
	}

	if err := h.apiKeyService.DeleteKey(id); err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(models.SuccessResponse{Success: true})
}

// BatchDeleteKeys handles batch deletion
func (h *Handlers) BatchDeleteKeys(c *fiber.Ctx) error {
	var req models.BatchDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Invalid request"})
	}

	if len(req.IDs) == 0 {
		return c.Status(400).JSON(models.ErrorResponse{Error: "No IDs provided"})
	}

	result, err := h.apiKeyService.BatchDeleteKeys(req.IDs)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(result)
}

// AddKey adds a single API key
func (h *Handlers) AddKey(c *fiber.Ctx) error {
	var req struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Invalid request"})
	}

	if req.Key == "" {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Key is required"})
	}

	// Import as single key
	result, err := h.apiKeyService.ImportKeys([]string{req.Key})
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponse{Error: err.Error()})
	}

	if result.Success > 0 {
		return c.JSON(models.SuccessResponse{
			Success: true,
			Message: "Key added successfully",
		})
	}

	if result.Duplicates > 0 {
		return c.Status(400).JSON(models.ErrorResponse{Error: "Key already exists"})
	}

	return c.Status(500).JSON(models.ErrorResponse{Error: "Failed to add key"})
}
