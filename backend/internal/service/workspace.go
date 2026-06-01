package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/DowneyTech/prism/backend/internal/model"
	"github.com/DowneyTech/prism/backend/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidationError wraps user-facing validation failures so the handler
// can map them to 422 without exporting a sentinel for every message.
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

func newValidationErr(msg string) error { return &ValidationError{msg: msg} }

var (
	ErrWorkspaceNotFound     = errors.New("workspace not found")
	ErrSlugTaken             = errors.New("slug already taken")
	ErrForbidden             = errors.New("forbidden")
	ErrNotMember             = errors.New("not a member of this workspace")
	ErrMemberNotFound        = errors.New("member not found")
	ErrAlreadyMember         = errors.New("user is already a member")
	ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin")
	ErrInvalidRole           = errors.New("role must be admin, member, or viewer")
	ErrInvalidSlug           = errors.New("slug may only contain lowercase letters, numbers, and hyphens")
	ErrInvitationNotFound    = errors.New("invitation not found")
	ErrInvitationExpired     = errors.New("invitation expired or already accepted")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

const inviteTTL = 72 * time.Hour

type WorkspaceService struct {
	pool        *pgxpool.Pool
	workspaces  *repository.WorkspaceRepository
	members     *repository.WorkspaceMemberRepository
	invitations *repository.InvitationRepository
	users       *repository.UserRepository
	appBaseURL  string
}

func NewWorkspaceService(
	pool *pgxpool.Pool,
	workspaces *repository.WorkspaceRepository,
	members *repository.WorkspaceMemberRepository,
	invitations *repository.InvitationRepository,
	users *repository.UserRepository,
	appBaseURL string,
) *WorkspaceService {
	return &WorkspaceService{
		pool:        pool,
		workspaces:  workspaces,
		members:     members,
		invitations: invitations,
		users:       users,
		appBaseURL:  appBaseURL,
	}
}

func (s *WorkspaceService) Create(ctx context.Context, userID string, req model.CreateWorkspaceRequest) (*model.WorkspaceDetailResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, newValidationErr("workspace name is required")
	}

	slug := req.Slug
	if slug == "" {
		var err error
		slug, err = s.generateUniqueSlug(ctx, req.Name)
		if err != nil {
			return nil, err
		}
	} else {
		if !slugPattern.MatchString(slug) {
			return nil, ErrInvalidSlug
		}
	}

	if err := validateDeadline(req.DeadlineDay, req.DeadlineHour); err != nil {
		return nil, err
	}

	ws, err := s.workspaces.Create(ctx, req.Name, slug, userID, req.DeadlineDay, req.DeadlineHour)
	if err != nil {
		if errors.Is(err, repository.ErrSlugTaken) {
			return nil, ErrSlugTaken
		}
		return nil, err
	}

	if _, err := s.members.Add(ctx, ws.ID, userID, "admin", userID); err != nil {
		return nil, fmt.Errorf("add creator as admin: %w", err)
	}

	return s.buildDetail(ctx, ws, userID)
}

func (s *WorkspaceService) Get(ctx context.Context, userID, slug string) (*model.WorkspaceDetailResponse, error) {
	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}
	if _, err := s.requireMember(ctx, ws.ID, userID); err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, ws, userID)
}

func (s *WorkspaceService) Update(ctx context.Context, userID, slug string, req model.UpdateWorkspaceRequest) (*model.WorkspaceDetailResponse, error) {
	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}
	if err := s.requireAdmin(ctx, ws.ID, userID); err != nil {
		return nil, err
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return nil, newValidationErr("workspace name cannot be empty")
	}
	if err := validateDeadline(req.DeadlineDay, req.DeadlineHour); err != nil {
		return nil, err
	}
	updated, err := s.workspaces.Update(ctx, ws.ID, req.Name, req.DeadlineDay, req.DeadlineHour)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}
	return s.buildDetail(ctx, updated, userID)
}

func (s *WorkspaceService) ListMembers(ctx context.Context, userID, slug string) ([]model.WorkspaceMember, error) {
	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}
	if _, err := s.requireMember(ctx, ws.ID, userID); err != nil {
		return nil, err
	}
	return s.members.ListWithUsers(ctx, ws.ID)
}

func (s *WorkspaceService) UpdateMemberRole(ctx context.Context, userID, slug, memberID, role string) error {
	if !isValidRole(role) {
		return ErrInvalidRole
	}
	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return s.mapWorkspaceErr(err)
	}
	if err := s.requireAdmin(ctx, ws.ID, userID); err != nil {
		return err
	}
	target, err := s.members.FindByID(ctx, memberID)
	if err != nil {
		return s.mapMemberErr(err)
	}
	if target.WorkspaceID != ws.ID {
		return ErrMemberNotFound
	}
	if target.Role == "admin" && role != "admin" {
		count, err := s.members.CountAdmins(ctx, ws.ID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}
	return s.members.UpdateRole(ctx, memberID, role)
}

func (s *WorkspaceService) RemoveMember(ctx context.Context, userID, slug, memberID string) error {
	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return s.mapWorkspaceErr(err)
	}

	// Admin check happens before member lookup to prevent membership-existence enumeration
	if err := s.requireAdmin(ctx, ws.ID, userID); err != nil {
		// Allow self-removal without admin rights
		target, fetchErr := s.members.FindByID(ctx, memberID)
		if fetchErr != nil {
			return s.mapMemberErr(fetchErr)
		}
		if target.WorkspaceID != ws.ID || target.UserID != userID {
			return err // return original requireAdmin error
		}
		// Self-removal path — fall through to last-admin check below
		if target.Role == "admin" {
			count, cerr := s.members.CountAdmins(ctx, ws.ID)
			if cerr != nil {
				return cerr
			}
			if count <= 1 {
				return ErrCannotRemoveLastAdmin
			}
		}
		return s.members.Remove(ctx, memberID)
	}

	// Admin removing another member
	target, err := s.members.FindByID(ctx, memberID)
	if err != nil {
		return s.mapMemberErr(err)
	}
	if target.WorkspaceID != ws.ID {
		return ErrMemberNotFound
	}
	if target.Role == "admin" {
		count, err := s.members.CountAdmins(ctx, ws.ID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}
	return s.members.Remove(ctx, memberID)
}

func (s *WorkspaceService) Invite(ctx context.Context, userID, slug string, req model.InviteMemberRequest) (*model.InviteResponse, error) {
	if req.Email == "" || !isValidEmail(req.Email) {
		return nil, ErrInvalidEmail
	}
	if req.Role != "member" && req.Role != "viewer" {
		return nil, newValidationErr("invite role must be member or viewer")
	}

	ws, err := s.workspaces.FindBySlug(ctx, slug)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}
	if err := s.requireAdmin(ctx, ws.ID, userID); err != nil {
		return nil, err
	}

	// Check if the invited email is already a member
	existing, err := s.users.FindByEmail(ctx, req.Email)
	if err == nil {
		_, memberErr := s.members.FindByUserAndWorkspace(ctx, ws.ID, existing.ID)
		if memberErr == nil {
			return nil, ErrAlreadyMember
		}
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	expiresAt := time.Now().Add(inviteTTL)
	inv, err := s.invitations.Create(ctx, ws.ID, req.Email, token, expiresAt)
	if err != nil {
		return nil, err
	}

	return &model.InviteResponse{
		InviteURL: fmt.Sprintf("%s/invite/%s", s.appBaseURL, inv.Token),
		Email:     inv.Email,
		ExpiresAt: inv.ExpiresAt,
	}, nil
}

func (s *WorkspaceService) AcceptInvitation(ctx context.Context, userID, token string) (*model.WorkspaceDetailResponse, error) {
	inv, err := s.invitations.FindByToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrInvitationNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	if inv.AcceptedAt != nil || time.Now().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	// Verify the logged-in user's email matches the invitation target
	callerUser, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(callerUser.Email, inv.Email) {
		return nil, ErrForbidden
	}

	ws, err := s.workspaces.FindByID(ctx, inv.WorkspaceID)
	if err != nil {
		return nil, s.mapWorkspaceErr(err)
	}

	// Already a member — idempotent accept
	if _, err := s.members.FindByUserAndWorkspace(ctx, inv.WorkspaceID, userID); err == nil {
		_ = s.invitations.Accept(ctx, inv.ID)
		return s.buildDetail(ctx, ws, userID)
	}

	// Atomic: mark invitation accepted AND add member in one transaction
	if err := s.acceptInviteTx(ctx, inv.ID, inv.WorkspaceID, userID); err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, ws, userID)
}

// acceptInviteTx marks the invitation accepted and adds the user as a member
// in a single transaction so neither operation can succeed without the other.
func (s *WorkspaceService) acceptInviteTx(ctx context.Context, invID, workspaceID, userID string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE invitations SET accepted_at = NOW()
		 WHERE id = $1 AND accepted_at IS NULL AND expires_at > NOW()`,
		invID,
	)
	if err != nil {
		return fmt.Errorf("accept invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvitationExpired
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO workspace_members (workspace_id, user_id, role, invited_by)
		 VALUES ($1, $2, 'member', $2)
		 ON CONFLICT (workspace_id, user_id) DO NOTHING`,
		workspaceID, userID,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return tx.Commit(ctx)
}

// ── helpers ────────────────────────────────────────────────

func (s *WorkspaceService) buildDetail(ctx context.Context, ws *model.Workspace, userID string) (*model.WorkspaceDetailResponse, error) {
	members, err := s.members.ListWithUsers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	myRole := ""
	for _, m := range members {
		if m.UserID == userID {
			myRole = m.Role
			break
		}
	}
	return &model.WorkspaceDetailResponse{
		Workspace: *ws,
		Members:   members,
		MyRole:    myRole,
	}, nil
}

func (s *WorkspaceService) requireMember(ctx context.Context, workspaceID, userID string) (*model.WorkspaceMember, error) {
	m, err := s.members.FindByUserAndWorkspace(ctx, workspaceID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return nil, ErrNotMember
		}
		return nil, err
	}
	return m, nil
}

func (s *WorkspaceService) requireAdmin(ctx context.Context, workspaceID, userID string) error {
	m, err := s.requireMember(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if m.Role != "admin" {
		return ErrForbidden
	}
	return nil
}

// getMemberUserID resolves the target member's userID, verifying workspace ownership first.
func (s *WorkspaceService) getMemberUserID(ctx context.Context, workspaceID, callerID, memberID string) (string, error) {
	target, err := s.members.FindByID(ctx, memberID)
	if err != nil {
		return "", s.mapMemberErr(err)
	}
	if target.WorkspaceID != workspaceID {
		return "", ErrMemberNotFound
	}
	return target.UserID, nil
}

func (s *WorkspaceService) generateUniqueSlug(ctx context.Context, name string) (string, error) {
	base := slugify(name)
	if len(base) < 3 {
		base = "workspace"
	}
	for i := 0; i <= 10; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		exists, err := s.workspaces.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", newValidationErr("could not generate a unique slug — try providing a custom slug")
}

func (s *WorkspaceService) mapWorkspaceErr(err error) error {
	if errors.Is(err, repository.ErrWorkspaceNotFound) {
		return ErrWorkspaceNotFound
	}
	return err
}

func (s *WorkspaceService) mapMemberErr(err error) error {
	if errors.Is(err, repository.ErrMemberNotFound) {
		return ErrMemberNotFound
	}
	if errors.Is(err, repository.ErrAlreadyMember) {
		return ErrAlreadyMember
	}
	return err
}

func validateDeadline(day, hour *int) error {
	if day != nil && (*day < 0 || *day > 6) {
		return newValidationErr("deadline_day must be 0 (Sun) to 6 (Sat)")
	}
	if hour != nil && (*hour < 0 || *hour > 23) {
		return newValidationErr("deadline_hour must be 0 to 23")
	}
	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isValidRole(r string) bool {
	return r == "admin" || r == "member" || r == "viewer"
}
