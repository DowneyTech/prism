package model

import "time"

type Workspace struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	CreatedBy    string    `json:"created_by"`
	DeadlineDay  *int      `json:"deadline_day,omitempty"`
	DeadlineHour *int      `json:"deadline_hour,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type WorkspaceMember struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
}

type Invitation struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Email       string     `json:"email"`
	Token       string     `json:"-"` // never serialised to clients
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
}

// --- Request types ---

type CreateWorkspaceRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug,omitempty"`
	DeadlineDay  *int   `json:"deadline_day,omitempty"`
	DeadlineHour *int   `json:"deadline_hour,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name         *string `json:"name,omitempty"`
	DeadlineDay  *int    `json:"deadline_day"` // explicit null clears the field
	DeadlineHour *int    `json:"deadline_hour"`
}

type InviteMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"` // "member" | "viewer"
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

// --- Response types ---

type WorkspaceDetailResponse struct {
	Workspace
	Members []WorkspaceMember `json:"members"`
	MyRole  string            `json:"my_role"`
}

type InviteResponse struct {
	InviteURL string    `json:"invite_url"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}
