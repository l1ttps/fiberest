package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fiberest/internal/common/pagination"
	"fiberest/internal/common/types"
	"fiberest/internal/models"
	"fiberest/internal/modules/users/dto"

	"github.com/google/uuid"
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
	CreateUser(ctx context.Context, email string, name string, role models.UserRole) (*models.User, error)
	CreateUserWithPassword(ctx context.Context, email string, password string, name string, role models.UserRole) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, id string, req dto.UpdateUserRequest) (*models.User, error)
	UpdateMyProfile(ctx context.Context, userID string, req dto.UpdateMyProfileRequest) (*models.User, error)
	SetPassword(ctx context.Context, userID string, password string) error
	DeleteUser(ctx context.Context, id string) error
	BanUser(ctx context.Context, id string, reason string, until *time.Time) error
	UnbanUser(ctx context.Context, id string) error
	GetManyUsers(ctx context.Context, req dto.GetManyUsersRequest) (*types.GetManyResponse[dto.UserResponse], error)
}

// service handles user-related business logic and responses
type service struct {
	repo Repository
}

// NewService creates a new user service instance
func NewService(repo Repository) UserService {
	return &service{
		repo: repo,
	}
}

// CreateUser creates a new user with specified role
func (s *service) CreateUser(ctx context.Context, email string, name string, role models.UserRole) (*models.User, error) {
	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check if user exists: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	user := &models.User{
		Email: email,
		Name:  name,
		Role:  role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// CreateUserWithPassword creates a new user with specified role and password
func (s *service) CreateUserWithPassword(ctx context.Context, email string, password string, name string, role models.UserRole) (*models.User, error) {
	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check if user exists: %w", err)
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email: email,
		Name:  name,
		Role:  role,
	}
	account := &models.Account{
		Type:      models.AccountTypeEmail,
		Password:  hashedPassword,
		IsPrimary: true,
	}

	if err := s.repo.CreateWithAccount(ctx, user, account); err != nil {
		return nil, fmt.Errorf("failed to create user with account: %w", err)
	}

	return user, nil
}

// FindByEmail finds a user by their email address
func (s *service) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.repo.FindByEmail(ctx, email)
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
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

// GetManyUsers retrieves a paginated list of users with total count.
func (s *service) GetManyUsers(ctx context.Context, req dto.GetManyUsersRequest) (*types.GetManyResponse[dto.UserResponse], error) {
	offset := (req.Page - 1) * req.Limit
	users, total, err := s.repo.List(ctx, req.Limit, offset, req.Role, req.Search)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	userResponses := make([]dto.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = dto.UserResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			Name:      user.Name,
			Role:      string(user.Role),
			BanReason: user.BanReason,
			BanUntil:  user.BanUntil,
		}
	}

	response := pagination.GetManyResponse(req.GetManyRequest, userResponses, total)

	return &response, nil
}

// UpdateMyProfile updates the current user's own profile information
func (s *service) UpdateMyProfile(ctx context.Context, userID string, req dto.UpdateMyProfileRequest) (*models.User, error) {
	user, err := s.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Name = req.Name

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information by ID
func (s *service) UpdateUser(ctx context.Context, id string, req dto.UpdateUserRequest) (*models.User, error) {
	if req.Role != "" {
		if req.Role != string(models.RoleAdmin) && req.Role != string(models.RoleUser) {
			return nil, fmt.Errorf("invalid role: %s", req.Role)
		}
	}

	user, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Email != "" {
		exists, err := s.repo.ExistsByEmailExcludingID(ctx, req.Email, id)
		if err != nil {
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if exists {
			return nil, ErrUserAlreadyExists
		}
		user.Email = req.Email
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Role != "" {
		user.Role = models.UserRole(req.Role)
	}

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// DeleteUser removes a user by ID
func (s *service) DeleteUser(ctx context.Context, id string) error {
	_, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// BanUser bans a user by ID with a reason and optional expiration time
func (s *service) BanUser(ctx context.Context, id string, reason string, until *time.Time) error {
	user, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}

	user.BanReason = reason
	user.BanUntil = until

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}

	return nil
}

// UnbanUser lifts the ban from a user
func (s *service) UnbanUser(ctx context.Context, id string) error {
	user, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}

	user.BanReason = ""
	user.BanUntil = nil

	if err := s.repo.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("failed to unban user: %w", err)
	}

	return nil
}

// hashPassword hashes a plain text password using bcrypt
func (s *service) hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// SetPassword sets a new password for a user by creating/updating their EMAIL account
func (s *service) SetPassword(ctx context.Context, userID string, password string) error {
	_, err := s.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	account, err := s.repo.FindAccount(ctx, userUUID, models.AccountTypeEmail)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		account = &models.Account{
			UserID:    userUUID,
			Type:      models.AccountTypeEmail,
			Password:  hashedPassword,
			IsPrimary: true,
		}
		if err := s.repo.CreateAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to find account: %w", err)
	} else {
		account.Password = hashedPassword
		if err := s.repo.SaveAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to update account password: %w", err)
		}
	}

	return nil
}
