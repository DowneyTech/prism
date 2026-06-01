package handler

import (
	"errors"
	"net/http"

	"github.com/DowneyTech/prism/backend/internal/model"
	"github.com/DowneyTech/prism/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type authHandler struct {
	auth *service.AuthService
}

func (h *authHandler) signup(c echo.Context) error {
	var req model.SignupRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	resp, err := h.auth.Signup(c.Request().Context(), req)
	if err != nil {
		return toHTTPError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *authHandler) login(c echo.Context) error {
	var req model.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	resp, err := h.auth.Login(c.Request().Context(), req)
	if err != nil {
		return toHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *authHandler) logout(c echo.Context) error {
	// JWT is stateless; revocation is handled client-side by discarding the token.
	return c.NoContent(http.StatusNoContent)
}

func toHTTPError(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, service.ErrEmailTaken):
		return echo.NewHTTPError(http.StatusConflict, "email already registered")
	case errors.Is(err, service.ErrInvalidCredentials):
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, service.ErrWeakPassword),
		errors.Is(err, service.ErrInvalidEmail),
		errors.Is(err, service.ErrNameRequired):
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
}
