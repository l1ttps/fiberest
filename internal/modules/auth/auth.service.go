package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"fiberest/internal/configs"
	"fiberest/internal/models"
	"fiberest/internal/modules/auth/dto"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrAdminExists       = errors.New("admin user already exists")
	ErrInvalidCredential = errors.New("invalid email or password")
	ErrSessionNotFound   = errors.New("session not found")
	ErrWrongPassword     = errors.New("current password is incorrect")
	ErrAccountBanned     = errors.New("account is banned")
)

// shortSessionDuration applies to logins without "remember me".
const shortSessionDuration = 24 * time.Hour

// AuthService defines the business logic for authentication.
// Keep this surface minimal: anything not used outside the module stays private.
type AuthService interface {
	CreateAdmin(ctx context.Context, req dto.InitAdminRequest) (*dto.InitAdminResponse, error)
	Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, sessionToken string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, sessionToken string, req dto.ChangePasswordRequest) (*dto.ChangePasswordResponse, error)

	// Session management
	FindSessionBySessionId(ctx context.Context, sessionToken string) (*models.Session, error)
	FindSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
	FindSessionsByUserID(ctx context.Context, userID string, limit, page int) ([]models.Session, int64, error)
	DeleteSession(ctx context.Context, sessionToken string) error
	DeleteSessionByID(ctx context.Context, sessionID uuid.UUID) error
	UpdateExpiresSession(ctx context.Context, session *models.Session) error
}

type service struct {
	repo Repository
	cfg  *configs.Config
}

// NewService creates a new auth service instance.
func NewService(repo Repository, cfg *configs.Config) AuthService {
	return &service{repo: repo, cfg: cfg}
}

// hashPassword hashes a plain text password using the configured bcrypt cost.
func (s *service) hashPassword(password string) (string, error) {
	cost := bcrypt.DefaultCost
	if s.cfg != nil && s.cfg.Auth.BcryptCost >= bcrypt.MinCost && s.cfg.Auth.BcryptCost <= bcrypt.MaxCost {
		cost = s.cfg.Auth.BcryptCost
	}
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// verifyPassword checks if the provided password matches the stored hash
func (s *service) verifyPassword(hashedPassword string, plainPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword)) == nil
}

// sessionDuration returns the configured remember-me session length.
func (s *service) sessionDuration() time.Duration {
	if s.cfg != nil && s.cfg.Auth.SessionDuration > 0 {
		return s.cfg.Auth.SessionDuration
	}
	return 7 * 24 * time.Hour
}

// CreateAdmin creates a new admin user and returns response.
func (s *service) CreateAdmin(ctx context.Context, req dto.InitAdminRequest) (*dto.InitAdminResponse, error) {
	adminCount, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing admin: %w", err)
	}
	if adminCount > 0 {
		return nil, ErrAdminExists
	}

	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email: req.Email,
		Name:  "Admin",
		Role:  models.RoleAdmin,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	account := &models.Account{
		UserID:    user.ID,
		Type:      models.AccountTypeEmail,
		Password:  hashedPassword,
		IsPrimary: true,
	}

	if err := s.repo.CreateAccount(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create admin account: %w", err)
	}

	return &dto.InitAdminResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Role:  string(user.Role),
	}, nil
}

// findUserByEmail finds a user by their email address
func (s *service) findUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

// generateSessionToken creates a cryptographically secure random token
func (s *service) generateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// createSession creates a new session for a user
func (s *service) createSession(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string, rememberMe bool) (*models.Session, error) {
	sessionToken, err := s.generateSessionToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(shortSessionDuration)
	if rememberMe {
		expiresAt = time.Now().Add(s.sessionDuration())
	}

	session := &models.Session{
		SessionToken: sessionToken,
		UserID:       userID,
		ExpiresAt:    expiresAt,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		RememberMe:   rememberMe,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// FindSessionBySessionId finds a session by its token
func (s *service) FindSessionBySessionId(ctx context.Context, sessionToken string) (*models.Session, error) {
	session, err := s.repo.FindSessionByToken(ctx, sessionToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	return session, nil
}

// DeleteSession removes a session by its token
func (s *service) DeleteSession(ctx context.Context, sessionToken string) error {
	if err := s.repo.DeleteSessionByToken(ctx, sessionToken); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// FindSessionByID finds a session by its primary key ID
func (s *service) FindSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	session, err := s.repo.FindSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	return session, nil
}

// DeleteSessionByID removes a session by its primary key ID
func (s *service) DeleteSessionByID(ctx context.Context, sessionID uuid.UUID) error {
	if err := s.repo.DeleteSessionByID(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// Login authenticates user and creates a session.
func (s *service) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	user, err := s.findUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, err
	}

	if user.BanUntil != nil && time.Now().Before(*user.BanUntil) {
		return nil, fmt.Errorf("%w: %s", ErrAccountBanned, user.BanReason)
	}

	account, err := s.repo.FindAccount(ctx, user.ID, models.AccountTypeEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	if !s.verifyPassword(account.Password, req.Password) {
		return nil, ErrInvalidCredential
	}

	session, err := s.createSession(ctx, user.ID, ipAddress, userAgent, req.Remember)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Message:      "Login successful",
		SessionToken: session.SessionToken,
		ExpiresAt:    session.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// Logout removes the current session
func (s *service) Logout(ctx context.Context, sessionToken string) error {
	return s.DeleteSession(ctx, sessionToken)
}

// ChangePassword verifies the current password, stores the new one and revokes
// every other session so a stolen cookie dies with the old password.
func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, sessionToken string, req dto.ChangePasswordRequest) (*dto.ChangePasswordResponse, error) {
	account, err := s.repo.FindAccount(ctx, userID, models.AccountTypeEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	if !s.verifyPassword(account.Password, req.CurrentPassword) {
		return nil, ErrWrongPassword
	}

	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateAccountPassword(ctx, account, hashedPassword); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.repo.DeleteSessionsByUserIDExceptToken(ctx, userID, sessionToken); err != nil {
		return nil, fmt.Errorf("failed to revoke other sessions: %w", err)
	}

	return &dto.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

// UpdateExpiresSession extends a remember-me session once per day.
func (s *service) UpdateExpiresSession(ctx context.Context, session *models.Session) error {
	if !session.RememberMe || !session.IsValid() {
		return nil
	}

	now := time.Now()
	if now.Sub(session.UpdatedAt) < 24*time.Hour {
		return nil
	}

	newExpiresAt := session.ExpiresAt.Add(s.sessionDuration())

	session.ExpiresAt = newExpiresAt
	session.UpdatedAt = now

	if err := s.repo.UpdateSession(ctx, session, map[string]any{
		"expires_at": newExpiresAt,
		"updated_at": now,
	}); err != nil {
		return fmt.Errorf("failed to update session expiration: %w", err)
	}

	return nil
}

// FindSessionsByUserID retrieves sessions for a specific user with pagination
func (s *service) FindSessionsByUserID(ctx context.Context, userID string, limit, page int) ([]models.Session, int64, error) {
	offset := (page - 1) * limit
	sessions, total, err := s.repo.FindSessionsByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sessions: %w", err)
	}
	return sessions, total, nil
}
