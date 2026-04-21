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
	"fiberest/internal/modules/auth/dto"
	"fiberest/internal/modules/auth/models"
	usermodels "fiberest/internal/modules/users/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrAdminExists       = errors.New("admin user already exists")
	ErrInvalidCredential = errors.New("invalid email or password")
	ErrSessionNotFound   = errors.New("session not found")
)

// AuthService defines the business logic for authentication
type AuthService interface {
	CreateAdmin(ctx context.Context, req dto.InitAdminRequest) (*dto.InitAdminResponse, error)
	Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, sessionToken string) error
	FindByEmail(ctx context.Context, email string) (*usermodels.User, error)
	FindByID(ctx context.Context, id string) (*usermodels.User, error)
	VerifyPassword(hashedPassword string, plainPassword string) bool

	// Session management
	CreateSession(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string) (*models.Session, error)
	FindValidSession(ctx context.Context, sessionToken string) (*models.Session, error)
	DeleteSession(ctx context.Context, sessionToken string) error
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
	if err := s.getDB(ctx).Model(&usermodels.User{}).Where("role = ?", usermodels.RoleAdmin).Count(&adminCount).Error; err != nil {
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
	user := &usermodels.User{
		Email: req.Email,
		Name:  "Admin",
		Role:  usermodels.RoleAdmin,
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
func (s *service) FindByEmail(ctx context.Context, email string) (*usermodels.User, error) {
	var user usermodels.User
	if err := s.getDB(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// FindByID finds a user by their ID
func (s *service) FindByID(ctx context.Context, id string) (*usermodels.User, error) {
	var user usermodels.User
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
func (s *service) CreateSession(ctx context.Context, userID uuid.UUID, ipAddress, userAgent string) (*models.Session, error) {
	sessionToken, err := s.generateSessionToken()
	if err != nil {
		return nil, err
	}

	session := &models.Session{
		SessionToken: sessionToken,
		UserID:       userID,
		ExpiresAt:    time.Now().Add(constants.SessionDuration),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := s.getDB(ctx).Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// FindValidSession finds a session by token and checks if it's still valid
func (s *service) FindValidSession(ctx context.Context, sessionToken string) (*models.Session, error) {
	var session models.Session
	if err := s.getDB(ctx).Where("session_token = ?", sessionToken).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if !session.IsValid() {
		// Session expired, delete it
		s.DeleteSession(ctx, sessionToken)
		return nil, errors.New("session expired")
	}

	return &session, nil
}

// DeleteSession removes a session by its token
func (s *service) DeleteSession(ctx context.Context, sessionToken string) error {
	if err := s.getDB(ctx).Where("session_token = ?", sessionToken).Delete(&models.Session{}).Error; err != nil {
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

	// Create new session
	session, err := s.CreateSession(ctx, user.ID, ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &dto.LoginResponse{
		Message:      "Login successful",
		SessionToken: session.SessionToken,
	}, nil
}

// Logout removes the current session
func (s *service) Logout(ctx context.Context, sessionToken string) error {
	return s.DeleteSession(ctx, sessionToken)
}
