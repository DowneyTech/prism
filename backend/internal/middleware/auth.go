package middleware

import (
	"net/http"
	"strings"

	"github.com/DowneyTech/prism/backend/internal/service"
	"github.com/labstack/echo/v4"
)

const ContextKeyUserID = "user_id"
const ContextKeyEmail = "user_email"

func JWT(authSvc *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid authorization header")
			}

			claims, err := authSvc.ParseToken(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			if claims.UserID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
			}
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyEmail, claims.Email)
			return next(c)
		}
	}
}
