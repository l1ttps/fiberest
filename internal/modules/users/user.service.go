package users

import (
	"context"
	"errors"
	"fmt"
	"math"

	"fiberest/internal/common/types"
	"fiberest/internal/database"
	"fiberest/internal/modules/users/dto"
	"fiberest/internal/modules/users/models"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrAdminExists       = errors.New("admin user already exists")
)

// UserService defines the business logic for user management
type UserService interface {
	CreateUser(ctx context.Context, email string, name string, role models.UserRole) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	GetManyUsers(ctx context.Context, req dto.GetManyUsersRequest) (*types.GetManyResponse[dto.UserResponse], error)
}

// service handles user-related business logic and responses
type service struct {
	dbService *database.DatabaseService
}

// NewService creates a new user service instance
func NewService(dbService *database.DatabaseService) UserService {
	return &service{
		dbService: dbService,
	}
}

// getDB returns the GORM database instance with context
func (s *service) getDB(ctx context.Context) *gorm.DB {
	return s.dbService.GetDB().WithContext(ctx)
}

// CreateUser creates a new user with specified role
func (s *service) CreateUser(ctx context.Context, email string, name string, role models.UserRole) (*models.User, error) {
	// Check if email already exists
	var exists bool
	if err := s.getDB(ctx).Model(&models.User{}).Select("count(*) > 0").Where("email = ?", email).Find(&exists).Error; err != nil {
		return nil, fmt.Errorf("failed to check if user exists: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Create the user
	user := &models.User{
		Email: email,
		Name:  name,
		Role:  role,
	}

	if err := s.getDB(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
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
