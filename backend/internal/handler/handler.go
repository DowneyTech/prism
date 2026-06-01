package handler

import (
	"net/http"

	"github.com/DowneyTech/prism/backend/internal/config"
	appmiddleware "github.com/DowneyTech/prism/backend/internal/middleware"
	"github.com/DowneyTech/prism/backend/internal/repository"
	"github.com/DowneyTech/prism/backend/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

func Register(e *echo.Echo, pool *pgxpool.Pool, cfg *config.Config) {
	userRepo := repository.NewUserRepository(pool)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public routes — no JWT required
	auth := &authHandler{auth: authSvc}
	a := e.Group("/api/auth")
	a.POST("/signup", auth.signup)
	a.POST("/login", auth.login)
	a.POST("/logout", auth.logout)

	// Protected routes — JWT required
	// All subsequent feature handlers are registered on this group.
	_ = e.Group("/api", appmiddleware.JWT(authSvc))
}
