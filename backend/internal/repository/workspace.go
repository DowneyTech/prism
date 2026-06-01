package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DowneyTech/prism/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSlugTaken          = errors.New("slug already taken")
	ErrWorkspaceNotFound  = errors.New("workspace not found")
	ErrMemberNotFound     = errors.New("member not found")
	ErrAlreadyMember      = errors.New("user is already a member")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation expired or already accepted")
)

// ── Workspace ──────────────────────────────────────────────

type WorkspaceRepository struct {
	pool *pgxpool.Pool
}

func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{pool: pool}
}

func (r *WorkspaceRepository) Create(ctx context.Context, name, slug, createdBy string, deadlineDay, deadlineHour *int) (*model.Workspace, error) {
	var w model.Workspace
	err := r.pool.QueryRow(ctx,
		`INSERT INTO workspaces (name, slug, created_by, deadline_day, deadline_hour)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, slug, created_by, deadline_day, deadline_hour, created_at`,
		name, slug, createdBy, deadlineDay, deadlineHour,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedBy, &w.DeadlineDay, &w.DeadlineHour, &w.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrSlugTaken
		}
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	return &w, nil
}

func (r *WorkspaceRepository) FindBySlug(ctx context.Context, slug string) (*model.Workspace, error) {
	var w model.Workspace
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, created_by, deadline_day, deadline_hour, created_at
		 FROM workspaces WHERE slug = $1`,
		slug,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedBy, &w.DeadlineDay, &w.DeadlineHour, &w.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("find workspace: %w", err)
	}
	return &w, nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, id string, name *string, deadlineDay, deadlineHour *int) (*model.Workspace, error) {
	var w model.Workspace
	err := r.pool.QueryRow(ctx,
		`UPDATE workspaces SET
		   name          = COALESCE($2, name),
		   deadline_day  = COALESCE($3, deadline_day),
		   deadline_hour = COALESCE($4, deadline_hour)
		 WHERE id = $1
		 RETURNING id, name, slug, created_by, deadline_day, deadline_hour, created_at`,
		id, name, deadlineDay, deadlineHour,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedBy, &w.DeadlineDay, &w.DeadlineHour, &w.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("update workspace: %w", err)
	}
	return &w, nil
}

func (r *WorkspaceRepository) FindByID(ctx context.Context, id string) (*model.Workspace, error) {
	var w model.Workspace
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, created_by, deadline_day, deadline_hour, created_at
		 FROM workspaces WHERE id = $1`,
		id,
	).Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedBy, &w.DeadlineDay, &w.DeadlineHour, &w.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, fmt.Errorf("find workspace by id: %w", err)
	}
	return &w, nil
}

func (r *WorkspaceRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1)`, slug,
	).Scan(&exists)
	return exists, err
}

// ── WorkspaceMember ────────────────────────────────────────

type WorkspaceMemberRepository struct {
	pool *pgxpool.Pool
}

func NewWorkspaceMemberRepository(pool *pgxpool.Pool) *WorkspaceMemberRepository {
	return &WorkspaceMemberRepository{pool: pool}
}

func (r *WorkspaceMemberRepository) Add(ctx context.Context, workspaceID, userID, role, invitedBy string) (*model.WorkspaceMember, error) {
	var m model.WorkspaceMember
	err := r.pool.QueryRow(ctx,
		`INSERT INTO workspace_members (workspace_id, user_id, role, invited_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, workspace_id, user_id, role, joined_at`,
		workspaceID, userID, role, invitedBy,
	).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyMember
		}
		return nil, fmt.Errorf("add member: %w", err)
	}
	return &m, nil
}

func (r *WorkspaceMemberRepository) ListWithUsers(ctx context.Context, workspaceID string) ([]model.WorkspaceMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT wm.id, wm.workspace_id, wm.user_id, wm.role, wm.joined_at,
		        u.name, u.email, u.avatar_url
		 FROM workspace_members wm
		 JOIN users u ON u.id = wm.user_id
		 WHERE wm.workspace_id = $1
		 ORDER BY wm.joined_at`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []model.WorkspaceMember
	for rows.Next() {
		var m model.WorkspaceMember
		if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt, &m.Name, &m.Email, &m.AvatarURL); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *WorkspaceMemberRepository) FindByUserAndWorkspace(ctx context.Context, workspaceID, userID string) (*model.WorkspaceMember, error) {
	var m model.WorkspaceMember
	err := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, user_id, role, joined_at
		 FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("find member: %w", err)
	}
	return &m, nil
}

func (r *WorkspaceMemberRepository) FindByID(ctx context.Context, memberID string) (*model.WorkspaceMember, error) {
	var m model.WorkspaceMember
	err := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, user_id, role, joined_at
		 FROM workspace_members WHERE id = $1`,
		memberID,
	).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("find member by id: %w", err)
	}
	return &m, nil
}

func (r *WorkspaceMemberRepository) UpdateRole(ctx context.Context, memberID, role string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE workspace_members SET role = $2 WHERE id = $1`,
		memberID, role,
	)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *WorkspaceMemberRepository) Remove(ctx context.Context, memberID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM workspace_members WHERE id = $1`, memberID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *WorkspaceMemberRepository) CountAdmins(ctx context.Context, workspaceID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1 AND role = 'admin'`,
		workspaceID,
	).Scan(&count)
	return count, err
}

// ── Invitation ─────────────────────────────────────────────

type InvitationRepository struct {
	pool *pgxpool.Pool
}

func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

func (r *InvitationRepository) Create(ctx context.Context, workspaceID, email, token string, expiresAt time.Time) (*model.Invitation, error) {
	var inv model.Invitation
	err := r.pool.QueryRow(ctx,
		`INSERT INTO invitations (workspace_id, email, token, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, workspace_id, email, token, expires_at, accepted_at`,
		workspaceID, email, token, expiresAt,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Token, &inv.ExpiresAt, &inv.AcceptedAt)
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}
	return &inv, nil
}

func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*model.Invitation, error) {
	var inv model.Invitation
	err := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, email, token, expires_at, accepted_at
		 FROM invitations WHERE token = $1`,
		token,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Token, &inv.ExpiresAt, &inv.AcceptedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, fmt.Errorf("find invitation: %w", err)
	}
	return &inv, nil
}

func (r *InvitationRepository) Accept(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE invitations SET accepted_at = NOW()
		 WHERE id = $1 AND accepted_at IS NULL AND expires_at > NOW()`,
		id,
	)
	if err != nil {
		return fmt.Errorf("accept invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvitationExpired
	}
	return nil
}
