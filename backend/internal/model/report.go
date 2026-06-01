package model

import "time"

type WeeklyReport struct {
	ID            string     `json:"id"`
	WorkspaceID   string     `json:"workspace_id"`
	UserID        string     `json:"user_id"`
	WeekStartDate string     `json:"week_start_date"` // YYYY-MM-DD (always a Monday)
	Done          *string    `json:"done,omitempty"`
	Blockers      *string    `json:"blockers,omitempty"`
	NextWeek      *string    `json:"next_week,omitempty"`
	Score         *int       `json:"score,omitempty"`
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
	UserName      string     `json:"user_name,omitempty"`
	UserEmail     string     `json:"user_email,omitempty"`
}

type SubmitReportRequest struct {
	WeekStartDate string  `json:"week_start_date,omitempty"` // optional; defaults to current week
	Done          *string `json:"done"`
	Blockers      *string `json:"blockers"`
	NextWeek      *string `json:"next_week"`
	Score         *int    `json:"score"`
}

type TeamReportsResponse struct {
	WeekStartDate string         `json:"week_start_date"`
	Reports       []WeeklyReport `json:"reports"`
	Submitted     int            `json:"submitted"`
	Total         int            `json:"total"`
}
