package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seoxpert/api/internal/cache"
	"github.com/seoxpert/api/internal/producer"
	"github.com/seoxpert/shared/config"
)

// Handler holds all shared dependencies injected once at startup.
// All handler methods hang off this struct — no global state.
type Handler struct {
	pool *pgxpool.Pool
	rdb  *cache.Client
	prod *producer.Producer
	cfg  *config.Config
}

func New(pool *pgxpool.Pool, rdb *cache.Client, prod *producer.Producer, cfg *config.Config) *Handler {
	return &Handler{
		pool: pool,
		rdb:  rdb,
		prod: prod,
		cfg:  cfg,
	}
}
