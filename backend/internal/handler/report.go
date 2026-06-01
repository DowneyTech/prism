package handler

import (
	"errors"
	"net/http"

	appmiddleware "github.com/DowneyTech/prism/backend/internal/middleware"
	"github.com/DowneyTech/prism/backend/internal/model"
	"github.com/DowneyTech/prism/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type reportHandler struct {
	svc *service.ReportService
}

func (h *reportHandler) submit(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	var req model.SubmitReportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	rep, err := h.svc.Submit(c.Request().Context(), userID, c.Param("slug"), req)
	if err != nil {
		return toReportHTTPError(err)
	}
	return c.JSON(http.StatusOK, rep)
}

func (h *reportHandler) getTeamReports(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	resp, err := h.svc.GetTeamReports(c.Request().Context(), userID, c.Param("slug"), c.QueryParam("week"))
	if err != nil {
		return toReportHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *reportHandler) getMyReports(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	reports, err := h.svc.GetMyReports(c.Request().Context(), userID, c.Param("slug"))
	if err != nil {
		return toReportHTTPError(err)
	}
	return c.JSON(http.StatusOK, reports)
}

func (h *reportHandler) getWeekReport(c echo.Context) error {
	userID, ok := c.Get(appmiddleware.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	resp, err := h.svc.GetWeekReport(c.Request().Context(), userID, c.Param("slug"), c.Param("week"))
	if err != nil {
		return toReportHTTPError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func toReportHTTPError(err error) *echo.HTTPError {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, ve.Error())
	}
	switch {
	case errors.Is(err, service.ErrWorkspaceNotFound),
		errors.Is(err, service.ErrReportNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrNotMember),
		errors.Is(err, service.ErrForbidden),
		errors.Is(err, service.ErrReportAccessDenied):
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrDeadlinePassed),
		errors.Is(err, service.ErrScoreOutOfRange):
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
}
