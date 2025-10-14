package services

import (
	"time"

	"github.com/droid-keyusage-go/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthService handles authentication
type AuthService struct {
	store         *storage.Storage
	adminPassword string
	jwtSecret     []byte
}

// NewAuthService creates a new auth service
func NewAuthService(store *storage.Storage, adminPassword string) *AuthService {
	// Generate a secret for JWT if not provided
	jwtSecret := []byte("your-secret-key-change-this-in-production")
	
	return &AuthService{
		store:         store,
		adminPassword: adminPassword,
		jwtSecret:     jwtSecret,
	}
}

// ValidatePassword checks if the password is correct
func (s *AuthService) ValidatePassword(password string) bool {
	// If no password is set, allow access
	if s.adminPassword == "" {
		return true
	}
	return password == s.adminPassword
}

// CreateSession creates a new session
func (s *AuthService) CreateSession() (string, error) {
	sessionID := uuid.New().String()
	
	session := &storage.Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}
	
	// Save to Redis with TTL
	err := s.store.SaveSession(session, 7*24*time.Hour)
	if err != nil {
		return "", err
	}
	
	return sessionID, nil
}

// ValidateSession checks if a session is valid
func (s *AuthService) ValidateSession(sessionID string) bool {
	if s.adminPassword == "" {
		return true // No auth required
	}
	
	if sessionID == "" {
		return false
	}
	
	session, err := s.store.GetSession(sessionID)
	if err != nil || session == nil {
		return false
	}
	
	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		_ = s.store.DeleteSession(sessionID)
		return false
	}
	
	return true
}

// DeleteSession removes a session
func (s *AuthService) DeleteSession(sessionID string) error {
	return s.store.DeleteSession(sessionID)
}

// IsAuthRequired checks if authentication is required
func (s *AuthService) IsAuthRequired() bool {
	return s.adminPassword != ""
}

// GenerateJWT creates a JWT token (alternative to session)
func (s *AuthService) GenerateJWT() (string, error) {
	claims := jwt.MapClaims{
		"authorized": true,
		"exp":        time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateJWT validates a JWT token
func (s *AuthService) ValidateJWT(tokenString string) bool {
	if s.adminPassword == "" {
		return true
	}
	
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	
	if err != nil || !token.Valid {
		return false
	}
	
	return true
}
