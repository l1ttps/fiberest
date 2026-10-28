package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"fiberest/internal/common/constants"
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

// AuthService defines the business logic for authentication
type AuthService interface {
	CreateAdmin(ctx context.Context, req dto.InitAdminRequest) (*dto.InitAdminResponse, error)
	Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, sessionToken string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) (*dto.ChangePasswordResponse, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	VerifyPassword(hashedPassword string, plainPassword string) bool

	// Session management
	CreateSession(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string, rememberMe bool) (*models.Session, error)
	FindSessionBySessionId(ctx context.Context, sessionToken string) (*models.Session, error)
	FindSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
	FindValidSession(ctx context.Context, sessionToken string) (*models.Session, error)
	FindSessionsByUserID(ctx context.Context, userID string, limit, page int) ([]models.Session, int64, error)
	DeleteSession(ctx context.Context, sessionToken string) error
	DeleteSessionByID(ctx context.Context, sessionID uuid.UUID) error
	UpdateExpiresSession(ctx context.Context, session *models.Session) error
}

type service struct {
	repo Repository
}

// NewService creates a new auth service instance
func NewService(repo Repository) AuthService {
	return &service{
		repo: repo,
	}
}

// hashPassword hashes a plain text password using bcrypt
func (s *service) hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the stored hash
func (s *service) VerifyPassword(hashedPassword string, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
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

	response := &dto.InitAdminResponse{
		ID:    user.ID.String(),
		Email: user.Email,
		Name:  user.Name,
		Role:  string(user.Role),
	}

	return response, nil
}

// FindByEmail finds a user by their email address
func (s *service) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

// FindByID finds a user by their ID
func (s *service) FindByID(ctx context.Context, id string) (*models.User, error) {
	user, err := s.repo.FindUserByID(ctx, id)
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
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session for a user
func (s *service) CreateSession(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string, rememberMe bool) (*models.Session, error) {
	sessionToken, err := s.generateSessionToken()
	if err != nil {
		return nil, err
	}

	var expiresAt time.Time
	if rememberMe {
		expiresAt = time.Now().Add(constants.SessionDuration)
	} else {
		expiresAt = time.Now().Add(24 * time.Hour)
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

// FindValidSession finds a session by token and checks if it's still valid
func (s *service) FindValidSession(ctx context.Context, sessionToken string) (*models.Session, error) {
	session, err := s.FindSessionBySessionId(ctx, sessionToken)
	if err != nil {
		return nil, err
	}

	if !session.IsValid() {
		_ = s.DeleteSession(ctx, sessionToken)
		return nil, errors.New("session expired")
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

// Login authenticates user and sets session cookie
func (s *service) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	user, err := s.FindByEmail(ctx, req.Email)
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
			return nil, errors.New("account not found for user")
		}
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	if !s.VerifyPassword(account.Password, req.Password) {
		return nil, ErrInvalidCredential
	}

	session, err := s.CreateSession(ctx, user.ID, ipAddress, userAgent, req.Remember)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
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

// ChangePassword changes the user's password after verifying the current password
func (s *service) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) (*dto.ChangePasswordResponse, error) {
	account, err := s.repo.FindAccount(ctx, userID, models.AccountTypeEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	if !s.VerifyPassword(account.Password, req.CurrentPassword) {
		return nil, ErrWrongPassword
	}

	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateAccountPassword(ctx, account, hashedPassword); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	return &dto.ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}

// UpdateExpiresSession extends the expiration of a remember-me session by 7 days
func (s *service) UpdateExpiresSession(ctx context.Context, session *models.Session) error {
	if !session.RememberMe {
		return nil
	}

	if !session.IsValid() {
		return nil
	}

	now := time.Now()

	if now.Sub(session.UpdatedAt) < 24*time.Hour {
		return nil
	}

	newExpiresAt := session.ExpiresAt.Add(constants.SessionDuration)

	session.ExpiresAt = newExpiresAt
	session.UpdatedAt = now

	if err := s.repo.UpdateSession(ctx, session, map[string]interface{}{
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
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return sessions, total, nil
}
