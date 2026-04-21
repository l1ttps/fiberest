package models

// UserRole defines the role of a user in the system
type UserRole string

const (
	// RoleAdmin has full system access
	RoleAdmin UserRole = "ADMIN"
	// RoleUser has limited access
	RoleUser UserRole = "USER"
)

// User represents a user account in the system.
// It extends BaseModel to inherit common fields like ID, CreatedAt, UpdatedAt.
type User struct {
	BaseModel

	// Email is the unique identifier for login
	// Must be unique and not null
	Email string `gorm:"uniqueIndex;not null;size:255" json:"email"`

	// Name is the display name of the user
	Name string `gorm:"not null;size:255" json:"name"`

	// Role determines user permissions (ADMIN or USER)
	Role UserRole `gorm:"not null;default:'USER'" json:"role"`
}

// TableName specifies the database table name for User model
func (User) TableName() string {
	return "users"
}
