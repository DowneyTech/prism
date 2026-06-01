package handler

import (
	"net/http"

	"github.com/DowneyTech/prism/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

func Register(e *echo.Echo, pool *pgxpool.Pool, cfg *config.Config) {
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	api := e.Group("/api")
	_ = api
	// 各ルートは以降のフェーズで追加
}
