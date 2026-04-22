package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"fiberest/internal/common/constants"
	"fiberest/internal/database"
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
	dbService *database.DatabaseService
}

// NewService creates a new auth service instance
func NewService(dbService *database.DatabaseService) AuthService {
	return &service{
		dbService: dbService,
	}
}

// getDB returns the GORM database instance with context
func (s *service) getDB(ctx context.Context) *gorm.DB {
	return s.dbService.GetDB().WithContext(ctx)
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
// This function can only be called when no admin exists in the system.
func (s *service) CreateAdmin(ctx context.Context, req dto.InitAdminRequest) (*dto.InitAdminResponse, error) {
	// Check if any admin already exists in the system
	var adminCount int64
	if err := s.getDB(ctx).Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount).Error; err != nil {
		return nil, fmt.Errorf("failed to check existing admin: %w", err)
	}
	if adminCount > 0 {
		return nil, ErrAdminExists
	}

	// Hash the password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create the user
	user := &models.User{
		Email: req.Email,
		Name:  "Admin",
		Role:  models.RoleAdmin,
	}

	if err := s.getDB(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	// Create the account for authentication
	account := &models.Account{
		UserID:    user.ID,
		Type:      models.AccountTypeEmail,
		Password:  hashedPassword,
		IsPrimary: true,
	}

	if err := s.getDB(ctx).Create(account).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin account: %w", err)
	}

	// Build response
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
	var user models.User
	if err := s.getDB(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// FindByID finds a user by their ID
func (s *service) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := s.getDB(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// generateSessionToken creates a cryptographically secure random token
func (s *service) generateSessionToken() (string, error) {
	bytes := make([]byte, 32) // 256-bit token
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

	// Determine expiration based on remember me flag
	var expiresAt time.Time
	if rememberMe {
		// Extended session duration for "remember me" (7 days)
		expiresAt = time.Now().Add(constants.SessionDuration)
	} else {
		// Standard session duration (1 day)
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

	if err := s.getDB(ctx).Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// FindSessionBySessionId finds a session by its token
func (s *service) FindSessionBySessionId(ctx context.Context, sessionToken string) (*models.Session, error) {
	var session models.Session
	if err := s.getDB(ctx).Preload("User").Where("session_token = ?", sessionToken).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	return &session, nil
}

// FindValidSession finds a session by token and checks if it's still valid
func (s *service) FindValidSession(ctx context.Context, sessionToken string) (*models.Session, error) {
	session, err := s.FindSessionBySessionId(ctx, sessionToken)
	if err != nil {
		return nil, err
	}

	if !session.IsValid() {
		// Session expired, delete it
		s.DeleteSession(ctx, sessionToken)
		return nil, errors.New("session expired")
	}

	return session, nil
}

// DeleteSession removes a session by its token
func (s *service) DeleteSession(ctx context.Context, sessionToken string) error {
	if err := s.getDB(ctx).Where("session_token = ?", sessionToken).Delete(&models.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// FindSessionByID finds a session by its primary key ID
func (s *service) FindSessionByID(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	var session models.Session
	if err := s.getDB(ctx).Where("id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	return &session, nil
}

// DeleteSessionByID removes a session by its primary key ID
func (s *service) DeleteSessionByID(ctx context.Context, sessionID uuid.UUID) error {
	if err := s.getDB(ctx).Where("id = ?", sessionID).Delete(&models.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// Login authenticates user and sets session cookie
func (s *service) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	// Find user by email
	user, err := s.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, err
	}

	// Find the user's EMAIL account
	var account models.Account
	if err := s.getDB(ctx).Where("user_id = ? AND type = ?", user.ID, models.AccountTypeEmail).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("account not found for user")
		}
		return nil, fmt.Errorf("failed to find account: %w", err)
	}

	// Verify password against account's hash
	if !s.VerifyPassword(account.Password, req.Password) {
		return nil, ErrInvalidCredential
	}

	// Create new session with remember me flag
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
	var account models.Account
	if err := s.getDB(ctx).Where("user_id = ? AND type = ?", userID, models.AccountTypeEmail).First(&account).Error; err != nil {
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

	if err := s.getDB(ctx).Model(&account).Update("password", hashedPassword).Error; err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	return &dto.ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}

// UpdateExpiresSession extends the expiration of a remember-me session by 7 days
// only when at least 24 hours have passed since the last update.
// This creates a rolling window:
// - Daily use → session never expires (each request extends by 7 days)
// - No use for 7+ days → session expires naturally
// - Maximum 1 DB update per day per session
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

	if err := s.getDB(ctx).Model(session).Updates(map[string]interface{}{
		"expires_at": newExpiresAt,
		"updated_at": now,
	}).Error; err != nil {
		return fmt.Errorf("failed to update session expiration: %w", err)
	}

	return nil
}

// FindSessionsByUserID retrieves sessions for a specific user with pagination
func (s *service) FindSessionsByUserID(ctx context.Context, userID string, limit, page int) ([]models.Session, int64, error) {
	var sessions []models.Session
	var total int64

	// Get total count
	if err := s.getDB(ctx).Model(&models.Session{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Fetch sessions with pagination
	if err := s.getDB(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find sessions: %w", err)
	}

	return sessions, total, nil
}
