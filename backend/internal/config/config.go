package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	Environment string
	DatabaseURL string
	RabbitMQURL string
	RedisURL    string
	AIServiceURL string
	JWTSecret   string
	LogLevel    string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://kanopt:kanopt@localhost:5432/kanopt?sslmode=disable"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		AIServiceURL: getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
