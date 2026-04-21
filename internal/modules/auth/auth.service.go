package auth

import (
	"context"
	"errors"
	"fmt"

	"fiberest/internal/common/auth"
	"fiberest/internal/common/constants"
	"fiberest/internal/database"
	"fiberest/internal/modules/auth/dto"
	usermodels "fiberest/internal/modules/users/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrAdminExists  = errors.New("admin user already exists")
)

// AuthService defines the business logic for authentication
type AuthService interface {
	CreateAdmin(ctx context.Context, req dto.InitAdminRequest) (*dto.InitAdminResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenResponse, error)
	VerifyPassword(hashedPassword string, plainPassword string) bool
	FindByEmail(ctx context.Context, email string) (*usermodels.User, error)
	FindByID(ctx context.Context, id string) (*usermodels.User, error)
}

type service struct {
	dbService  *database.DatabaseService
	jwtService *auth.JWTService
}

// NewService creates a new auth service instance
func NewService(dbService *database.DatabaseService, jwtService *auth.JWTService) AuthService {
	return &service{
		dbService:  dbService,
		jwtService: jwtService,
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
		Email:    req.Email,
		Name:     "Admin",
		Password: hashedPassword,
		Role:     usermodels.RoleAdmin,
	}

	if err := s.getDB(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
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

// generateTokenPair creates a pair of access and refresh tokens for a user
func (s *service) generateTokenPair(user *usermodels.User) (*dto.TokenResponse, error) {
	accessToken, err := s.jwtService.GenerateToken(user.ID.String(), string(user.Role), constants.AccessTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateToken(user.ID.String(), string(user.Role), constants.RefreshTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Login authenticates user and returns a token pair
func (s *service) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
	user, err := s.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, err
	}

	if !s.VerifyPassword(user.Password, req.Password) {
		return nil, fmt.Errorf("invalid email or password")
	}

	return s.generateTokenPair(user)
}

// RefreshToken validates a refresh token and returns a new token pair
func (s *service) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
	claims, err := s.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	user, err := s.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	return s.generateTokenPair(user)
}
