package users

import (
	"fmt"

	"fiberest/internal/database"
	"fiberest/internal/modules/users/dto"
	"fiberest/internal/modules/users/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Service handles user-related business logic and responses
type Service struct {
	dbService *database.DatabaseService
}

// NewService creates a new user service instance
func NewService(dbService *database.DatabaseService) *Service {
	return &Service{
		dbService: dbService,
	}
}

// getDB returns the GORM database instance
func (s *Service) getDB() *gorm.DB {
	return s.dbService.GetDB()
}

// hashPassword hashes a plain text password using bcrypt
func (s *Service) hashPassword(password string) (string, error) {
	// bcrypt.DefaultCost is 10, which provides a good balance between security and performance
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// CreateAdmin creates a new admin user and returns response
func (s *Service) CreateAdmin(req dto.InitAdminRequest) (*dto.InitAdminResponse, error) {
	// Check if email already exists
	var existingUser models.User
	if err := s.getDB().Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Hash the password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create the admin user
	user := &models.User{
		Email:    req.Email,
		Name:     req.Email, // Default name is email, can be updated later
		Password: hashedPassword,
		Role:     models.RoleAdmin,
	}

	if err := s.getDB().Create(user).Error; err != nil {
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

// CreateUser creates a new regular user
func (s *Service) CreateUser(email string, name string, password string) (*models.User, error) {
	// Check if email already exists
	var existingUser models.User
	if err := s.getDB().Where("email = ?", email).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
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
		Role:     models.RoleUser,
	}

	if err := s.getDB().Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// VerifyPassword checks if the provided password matches the stored hash
func (s *Service) VerifyPassword(hashedPassword string, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// FindByEmail finds a user by their email address
func (s *Service) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := s.getDB().Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// FindByID finds a user by their ID
func (s *Service) FindByID(id string) (*models.User, error) {
	var user models.User
	if err := s.getDB().Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}
