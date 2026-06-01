package main

import (
	"log"

	"github.com/DowneyTech/prism/backend/internal/config"
	"github.com/DowneyTech/prism/backend/internal/db"
	"github.com/DowneyTech/prism/backend/internal/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()
	if cfg.JWTSecret == "change-me-in-production" || cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set to a strong random value before starting")
	}

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("1M"))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{cfg.FrontendURL},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
	}))

	handler.Register(e, pool, cfg)

	log.Fatal(e.Start(":" + cfg.Port))
}
