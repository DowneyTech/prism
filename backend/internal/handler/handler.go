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

	wsRepo := repository.NewWorkspaceRepository(pool)
	memberRepo := repository.NewWorkspaceMemberRepository(pool)
	invRepo := repository.NewInvitationRepository(pool)
	wsSvc := service.NewWorkspaceService(pool, wsRepo, memberRepo, invRepo, userRepo, cfg.FrontendURL)

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
	p := e.Group("/api", appmiddleware.JWT(authSvc))

	ws := &workspaceHandler{ws: wsSvc}
	p.POST("/workspaces", ws.create)
	p.GET("/workspaces/:slug", ws.get)
	p.PUT("/workspaces/:slug", ws.update)
	p.GET("/workspaces/:slug/members", ws.listMembers)
	p.PUT("/workspaces/:slug/members/:id", ws.updateMemberRole)
	p.DELETE("/workspaces/:slug/members/:id", ws.removeMember)
	p.POST("/workspaces/:slug/invite", ws.invite)
	p.POST("/invitations/:token/accept", ws.acceptInvite)
}
