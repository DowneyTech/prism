package handler

import (
	"errors"
	"net/http"

	appmiddleware "github.com/DowneyTech/prism/backend/internal/middleware"
	"github.com/DowneyTech/prism/backend/internal/model"
	"github.com/DowneyTech/prism/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type workspaceHandler struct {
	ws *service.WorkspaceService
}

func (h *workspaceHandler) create(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	var req model.CreateWorkspaceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	resp, err := h.ws.Create(c.Request().Context(), userID, req)
	if err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *workspaceHandler) get(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	resp, err := h.ws.Get(c.Request().Context(), userID, c.Param("slug"))
	if err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *workspaceHandler) update(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	var req model.UpdateWorkspaceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	resp, err := h.ws.Update(c.Request().Context(), userID, c.Param("slug"), req)
	if err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *workspaceHandler) listMembers(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	members, err := h.ws.ListMembers(c.Request().Context(), userID, c.Param("slug"))
	if err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.JSON(http.StatusOK, members)
}

func (h *workspaceHandler) updateMemberRole(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	var req model.UpdateMemberRoleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := h.ws.UpdateMemberRole(c.Request().Context(), userID, c.Param("slug"), c.Param("id"), req.Role); err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *workspaceHandler) removeMember(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if err := h.ws.RemoveMember(c.Request().Context(), userID, c.Param("slug"), c.Param("id")); err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *workspaceHandler) invite(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	var req model.InviteMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	resp, err := h.ws.Invite(c.Request().Context(), userID, c.Param("slug"), req)
	if err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *workspaceHandler) acceptInvite(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	resp, err := h.ws.AcceptInvitation(c.Request().Context(), userID, c.Param("token"))
	if err != nil {
		return toWorkspaceHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func toWorkspaceHTTPError(err error) *echo.HTTPError {
	// ValidationError covers all user-facing validation failures
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, ve.Error())
	}
	switch {
	case errors.Is(err, service.ErrWorkspaceNotFound),
		errors.Is(err, service.ErrMemberNotFound),
		errors.Is(err, service.ErrInvitationNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrForbidden),
		errors.Is(err, service.ErrNotMember):
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrSlugTaken),
		errors.Is(err, service.ErrAlreadyMember):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrCannotRemoveLastAdmin),
		errors.Is(err, service.ErrInvalidRole),
		errors.Is(err, service.ErrInvalidSlug),
		errors.Is(err, service.ErrInvitationExpired),
		errors.Is(err, service.ErrInvalidEmail):
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
}
