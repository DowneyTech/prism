package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/DowneyTech/prism/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrReportNotFound = errors.New("report not found")

type ReportRepository struct {
	pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

// Upsert inserts or updates the report for (workspace, user, week).
// submitted_at is set only on first insert.
func (r *ReportRepository) Upsert(ctx context.Context, workspaceID, userID, weekStartDate string, done, blockers, nextWeek *string, score *int) (*model.WeeklyReport, error) {
	var rep model.WeeklyReport
	err := r.pool.QueryRow(ctx,
		`INSERT INTO weekly_reports
		   (workspace_id, user_id, week_start_date, done, blockers, next_week, score, submitted_at, updated_at)
		 VALUES ($1, $2, $3::date, $4, $5, $6, $7, NOW(), NOW())
		 ON CONFLICT (workspace_id, user_id, week_start_date) DO UPDATE SET
		   done        = EXCLUDED.done,
		   blockers    = EXCLUDED.blockers,
		   next_week   = EXCLUDED.next_week,
		   score       = EXCLUDED.score,
		   updated_at  = NOW()
		 RETURNING id, workspace_id, user_id, week_start_date::text,
		           done, blockers, next_week, score, submitted_at, updated_at`,
		workspaceID, userID, weekStartDate, done, blockers, nextWeek, score,
	).Scan(
		&rep.ID, &rep.WorkspaceID, &rep.UserID, &rep.WeekStartDate,
		&rep.Done, &rep.Blockers, &rep.NextWeek, &rep.Score,
		&rep.SubmittedAt, &rep.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert report: %w", err)
	}
	return &rep, nil
}

// GetTeamReports returns all reports (with user info) for a given workspace and week.
// Members without a submitted report are included as nil entries in callers if needed;
// here we only return rows that exist.
func (r *ReportRepository) GetTeamReports(ctx context.Context, workspaceID, weekStartDate string) ([]model.WeeklyReport, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT wr.id, wr.workspace_id, wr.user_id, wr.week_start_date::text,
		        wr.done, wr.blockers, wr.next_week, wr.score, wr.submitted_at, wr.updated_at,
		        u.name, u.email
		 FROM weekly_reports wr
		 JOIN users u ON u.id = wr.user_id
		 JOIN workspace_members wm ON wm.workspace_id = wr.workspace_id
		   AND wm.user_id = wr.user_id AND wm.role IN ('admin', 'member')
		 WHERE wr.workspace_id = $1 AND wr.week_start_date = $2::date
		 ORDER BY u.name`,
		workspaceID, weekStartDate,
	)
	if err != nil {
		return nil, fmt.Errorf("get team reports: %w", err)
	}
	defer rows.Close()

	reports := make([]model.WeeklyReport, 0)
	for rows.Next() {
		var rep model.WeeklyReport
		if err := rows.Scan(
			&rep.ID, &rep.WorkspaceID, &rep.UserID, &rep.WeekStartDate,
			&rep.Done, &rep.Blockers, &rep.NextWeek, &rep.Score,
			&rep.SubmittedAt, &rep.UpdatedAt,
			&rep.UserName, &rep.UserEmail,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

// GetUserReports returns all reports submitted by a user in a workspace, most recent first.
func (r *ReportRepository) GetUserReports(ctx context.Context, workspaceID, userID string) ([]model.WeeklyReport, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, workspace_id, user_id, week_start_date::text,
		        done, blockers, next_week, score, submitted_at, updated_at
		 FROM weekly_reports
		 WHERE workspace_id = $1 AND user_id = $2
		 ORDER BY week_start_date DESC`,
		workspaceID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get user reports: %w", err)
	}
	defer rows.Close()

	reports := make([]model.WeeklyReport, 0)
	for rows.Next() {
		var rep model.WeeklyReport
		if err := rows.Scan(
			&rep.ID, &rep.WorkspaceID, &rep.UserID, &rep.WeekStartDate,
			&rep.Done, &rep.Blockers, &rep.NextWeek, &rep.Score,
			&rep.SubmittedAt, &rep.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user report: %w", err)
		}
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

// GetReport returns the report for a specific user and week.
func (r *ReportRepository) GetReport(ctx context.Context, workspaceID, userID, weekStartDate string) (*model.WeeklyReport, error) {
	var rep model.WeeklyReport
	err := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, user_id, week_start_date::text,
		        done, blockers, next_week, score, submitted_at, updated_at
		 FROM weekly_reports
		 WHERE workspace_id = $1 AND user_id = $2 AND week_start_date = $3::date`,
		workspaceID, userID, weekStartDate,
	).Scan(
		&rep.ID, &rep.WorkspaceID, &rep.UserID, &rep.WeekStartDate,
		&rep.Done, &rep.Blockers, &rep.NextWeek, &rep.Score,
		&rep.SubmittedAt, &rep.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReportNotFound
		}
		return nil, fmt.Errorf("get report: %w", err)
	}
	return &rep, nil
}

// MemberCount returns the number of members (admin + member roles) in a workspace.
// Viewers are excluded since they don't submit reports.
func (r *ReportRepository) MemberCount(ctx context.Context, workspaceID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM workspace_members
		 WHERE workspace_id = $1 AND role IN ('admin', 'member')`,
		workspaceID,
	).Scan(&count)
	return count, err
}
