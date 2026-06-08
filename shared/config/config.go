package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all environment-based configuration.
// All fields are loaded once at startup — zero allocations at runtime.
type Config struct {
	// Server
	Port         string
	Env          string
	AllowOrigins string

	// Supabase (self-hosted)
	SupabaseURL       string
	SupabaseAnonKey   string
	SupabaseJWTSecret string
	DatabaseURL       string // postgres://user:pass@host:5432/db?pool_max_conns=20

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// ClickHouse
	ClickHouseAddr     string
	ClickHouseDB       string
	ClickHouseUser     string
	ClickHousePassword string

	// Redpanda (Kafka-compatible)
	RedpandaBrokers []string

	// Supabase Storage
	StorageBucket string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		Env:          getEnv("APP_ENV", "development"),
		AllowOrigins: getEnv("ALLOW_ORIGINS", "http://localhost:3000"),

		SupabaseURL:       mustEnv("SUPABASE_URL"),
		SupabaseAnonKey:   mustEnv("SUPABASE_ANON_KEY"),
		SupabaseJWTSecret: mustEnv("SUPABASE_JWT_SECRET"),
		DatabaseURL:       mustEnv("DATABASE_URL"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 1), // DB 1 = seoxpert (0 = shared)

		ClickHouseAddr:     getEnv("CLICKHOUSE_ADDR", "localhost:9000"),
		ClickHouseDB:       getEnv("CLICKHOUSE_DB", "seoxpert"),
		ClickHouseUser:     getEnv("CLICKHOUSE_USER", "default"),
		ClickHousePassword: getEnv("CLICKHOUSE_PASSWORD", ""),

		RedpandaBrokers: strings.Split(getEnv("REDPANDA_BROKERS", "localhost:9092"), ","),
		StorageBucket:   getEnv("STORAGE_BUCKET", "seoxpert"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustEnv panics at startup if a required env var is missing — fail fast.
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
