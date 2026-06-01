package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DowneyTech/prism/backend/internal/model"
	"github.com/DowneyTech/prism/backend/internal/repository"
)

var (
	ErrReportNotFound    = errors.New("report not found")
	ErrDeadlinePassed    = errors.New("submission deadline has passed")
	ErrScoreOutOfRange   = errors.New("score must be between 1 and 5")
	ErrReportAccessDenied = errors.New("viewers cannot submit reports")
)

type ReportService struct {
	workspaces *repository.WorkspaceRepository
	members    *repository.WorkspaceMemberRepository
	reports    *repository.ReportRepository
}

func NewReportService(
	workspaces *repository.WorkspaceRepository,
	members *repository.WorkspaceMemberRepository,
	reports *repository.ReportRepository,
) *ReportService {
	return &ReportService{
		workspaces: workspaces,
		members:    members,
		reports:    reports,
	}
}

// Submit upserts the current user's report for the given week.
// If week_start_date is omitted it defaults to the current week's Monday.
// Viewers may not submit.
// After the workspace deadline the submission is rejected.
func (s *ReportService) Submit(ctx context.Context, userID, slug string, req model.SubmitReportRequest) (*model.WeeklyReport, error) {
	if req.Score != nil && (*req.Score < 1 || *req.Score > 5) {
		return nil, ErrScoreOutOfRange
	}

	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}

	m, err := s.members.FindByUserAndWorkspace(ctx, ws.ID, userID)
	if err != nil {
		return nil, s.mapMemberErr(err)
	}
	if m.Role == "viewer" {
		return nil, ErrReportAccessDenied
	}

	weekStr := req.WeekStartDate
	if weekStr == "" {
		weekStr = currentWeekMonday()
	} else {
		// Normalise whatever date the client sends to its Monday
		weekStr, err = normaliseToMonday(weekStr)
		if err != nil {
			return nil, newValidationErr("week_start_date must be a valid date (YYYY-MM-DD)")
		}
	}

	if isDeadlinePassed(ws.DeadlineDay, ws.DeadlineHour, weekStr) {
		return nil, ErrDeadlinePassed
	}

	return s.reports.Upsert(ctx, ws.ID, userID, weekStr, req.Done, req.Blockers, req.NextWeek, req.Score)
}

// GetTeamReports returns all submitted reports for a workspace week.
// Any member (including viewers) may call this.
func (s *ReportService) GetTeamReports(ctx context.Context, userID, slug, weekStr string) (*model.TeamReportsResponse, error) {
	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}
	if _, err := s.members.FindByUserAndWorkspace(ctx, ws.ID, userID); err != nil {
		return nil, s.mapMemberErr(err)
	}

	if weekStr == "" {
		weekStr = currentWeekMonday()
	} else {
		weekStr, err = normaliseToMonday(weekStr)
		if err != nil {
			return nil, newValidationErr("week must be a valid date (YYYY-MM-DD)")
		}
	}

	reports, err := s.reports.GetTeamReports(ctx, ws.ID, weekStr)
	if err != nil {
		return nil, fmt.Errorf("get team reports: %w", err)
	}

	total, err := s.reports.MemberCount(ctx, ws.ID)
	if err != nil {
		return nil, fmt.Errorf("member count: %w", err)
	}

	return &model.TeamReportsResponse{
		WeekStartDate: weekStr,
		Reports:       reports,
		Submitted:     len(reports),
		Total:         total,
	}, nil
}

// GetMyReports returns all reports submitted by the calling user in the workspace.
func (s *ReportService) GetMyReports(ctx context.Context, userID, slug string) ([]model.WeeklyReport, error) {
	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}
	if _, err := s.members.FindByUserAndWorkspace(ctx, ws.ID, userID); err != nil {
		return nil, s.mapMemberErr(err)
	}
	return s.reports.GetUserReports(ctx, ws.ID, userID)
}

// GetWeekReport returns the team report for a specific week (alias of GetTeamReports).
// Kept separate so the handler can route :week vs ?week differently if needed.
func (s *ReportService) GetWeekReport(ctx context.Context, userID, slug, weekStr string) (*model.TeamReportsResponse, error) {
	return s.GetTeamReports(ctx, userID, slug, weekStr)
}

// ── date helpers ────────────────────────────────────────────

// currentWeekMonday returns the Monday of the current UTC week as YYYY-MM-DD.
func currentWeekMonday() string {
	return mondayOf(time.Now().UTC())
}

// normaliseToMonday parses dateStr (YYYY-MM-DD) and returns the Monday of that week.
func normaliseToMonday(dateStr string) (string, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", err
	}
	return mondayOf(t), nil
}

func mondayOf(t time.Time) string {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → treat as 7 so Monday offset is correct
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return monday.Format("2006-01-02")
}

// isDeadlinePassed reports whether the submission window for weekStr has closed.
// If the workspace has no deadline configured, submission is always open.
// The deadline is interpreted as: weekday `day` at hour `hour` UTC of the week
// that starts on `weekStr`.
func isDeadlinePassed(deadlineDay, deadlineHour *int, weekStr string) bool {
	if deadlineDay == nil || deadlineHour == nil {
		return false
	}

	monday, err := time.Parse("2006-01-02", weekStr)
	if err != nil {
		return false
	}

	// deadline_day: 0=Sun, 1=Mon, …, 6=Sat — same as time.Weekday
	// We want the deadline within the same calendar week that starts on `monday`.
	// Convert Sunday (0) to 7 so the arithmetic stays within Mon–Sun.
	targetDay := *deadlineDay
	if targetDay == 0 {
		targetDay = 7
	}
	// monday is day 1; offset from monday to the deadline day
	deadline := monday.AddDate(0, 0, targetDay-1).Add(time.Duration(*deadlineHour) * time.Hour)
	return time.Now().UTC().After(deadline)
}

// ── error mappers ───────────────────────────────────────────

func (s *ReportService) mapWorkspaceErr(err error) error {
	if errors.Is(err, repository.ErrWorkspaceNotFound) {
		return ErrWorkspaceNotFound
	}
	return err
}

func (s *ReportService) mapMemberErr(err error) error {
	if errors.Is(err, repository.ErrMemberNotFound) {
		return ErrNotMember
	}
	return err
}
