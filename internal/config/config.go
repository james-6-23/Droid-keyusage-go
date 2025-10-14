package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port string
	Env  string

	// Redis
	RedisURL      string
	RedisPassword string
	RedisDB       int

	// Auth
	AdminPassword string
	SessionTTL    time.Duration

	// Worker Pool
	MaxWorkers int
	QueueSize  int

	// HTTP Client
	HTTPTimeout time.Duration
	MaxRetries  int

	// Cache
	CacheTTL       time.Duration
	LocalCacheSize int

	// Rate Limiting
	RateLimit      int
	RateLimitBurst int
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		SessionTTL:    getEnvAsDuration("SESSION_TTL", 7*24*time.Hour),

		MaxWorkers: getEnvAsInt("MAX_WORKERS", 100),
		QueueSize:  getEnvAsInt("QUEUE_SIZE", 10000),

		HTTPTimeout: getEnvAsDuration("HTTP_TIMEOUT", 30*time.Second),
		MaxRetries:  getEnvAsInt("MAX_RETRIES", 3),

		CacheTTL:       getEnvAsDuration("CACHE_TTL", 5*time.Minute),
		LocalCacheSize: getEnvAsInt("LOCAL_CACHE_SIZE", 1000),

		RateLimit:      getEnvAsInt("RATE_LIMIT", 100),
		RateLimitBurst: getEnvAsInt("RATE_LIMIT_BURST", 200),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}
