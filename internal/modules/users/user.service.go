package users

import (
	"context"
	"errors"
	"fmt"
	"math"

	"fiberest/internal/common/auth"
	"fiberest/internal/common/constants"
	"fiberest/internal/common/types"
	"fiberest/internal/database"
	"fiberest/internal/modules/users/dto"
	"fiberest/internal/modules/users/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrAdminExists       = errors.New("admin user already exists")
)

// UserService defines the business logic for user management
type UserService interface {
	CreateAdmin(ctx context.Context, req dto.InitAdminRequest) (*dto.InitAdminResponse, error)
	CreateUser(ctx context.Context, email string, name string, password string, role models.UserRole) (*models.User, error)
	VerifyPassword(hashedPassword string, plainPassword string) bool
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	GetManyUsers(ctx context.Context, req dto.GetManyUsersRequest) (*types.GetManyResponse[dto.UserResponse], error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenResponse, error)
}

// service handles user-related business logic and responses
type service struct {
	dbService  *database.DatabaseService
	jwtService *auth.JWTService
}

// NewService creates a new user service instance
func NewService(dbService *database.DatabaseService, jwtService *auth.JWTService) UserService {
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
	// bcrypt.DefaultCost is 10, which provides a good balance between security and performance
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
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

	// Reuse CreateUser to create admin user
	user, err := s.CreateUser(ctx, req.Email, "Admin", req.Password, models.RoleAdmin)

	if err != nil {
		return nil, err
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

// CreateUser creates a new user with specified role
func (s *service) CreateUser(ctx context.Context, email string, name string, password string, role models.UserRole) (*models.User, error) {
	// Check if email already exists
	var exists bool
	if err := s.getDB(ctx).Model(&models.User{}).Select("count(*) > 0").Where("email = ?", email).Find(&exists).Error; err != nil {
		return nil, fmt.Errorf("failed to check if user exists: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Hash the password
	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create the user
	user := &models.User{
		Email:    email,
		Name:     name,
		Password: hashedPassword,
		Role:     role,
	}

	if err := s.getDB(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// VerifyPassword checks if the provided password matches the stored hash
func (s *service) VerifyPassword(hashedPassword string, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
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

// GetManyUsers retrieves a paginated list of users with total count.
// It returns a GetManyResponse containing users, pagination info and total count.
// Supports optional filtering by role and search query (name or email).
func (s *service) GetManyUsers(ctx context.Context, req dto.GetManyUsersRequest) (*types.GetManyResponse[dto.UserResponse], error) {
	// Calculate offset
	offset := (req.Page - 1) * req.Limit

	// Build base query with optional filters
	query := s.getDB(ctx).Model(&models.User{})

	// Apply role filter if provided
	if req.Role != "" {
		query = query.Where("role = ?", req.Role)
	}

	// Apply search filter if provided (search in name and email, case-insensitive)
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern)
	}

	// Get total count with filters applied
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Fetch users with pagination and filters
	var users []models.User
	if err := query.
		Limit(req.Limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	// Map to response DTOs
	userResponses := make([]dto.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = dto.UserResponse{
			ID:    user.ID.String(),
			Email: user.Email,
			Name:  user.Name,
			Role:  string(user.Role),
		}
	}

	// Calculate hasNextPage
	totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
	hasNextPage := req.Page < totalPages

	// Build response
	response := &types.GetManyResponse[dto.UserResponse]{
		Data:        userResponses,
		Limit:       req.Limit,
		Page:        req.Page,
		HasNextPage: hasNextPage,
		Total:       total,
	}

	return response, nil
}

// generateTokenPair creates a pair of access and refresh tokens for a user
func (s *service) generateTokenPair(user *models.User) (*dto.TokenResponse, error) {
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
