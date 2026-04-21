package models

import (
	commonModels "fiberest/internal/common/models"
	userModels "fiberest/internal/modules/users/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountType defines the authentication method type
type AccountType string

const (
	AccountTypeEmail  AccountType = "EMAIL"
	AccountTypeGoogle AccountType = "GOOGLE"
	AccountTypeGitHub AccountType = "GITHUB"
)

// Account represents an authentication account linked to a user.
// A user can have multiple accounts (email, google, github, etc.)
type Account struct {
	commonModels.BaseModel

	// UserID is the foreign key to the user
	UserID uuid.UUID `gorm:"not null;index" json:"userId"`

	// User relationship (many accounts belong to one user)
	User userModels.User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`

	// Type indicates the authentication provider
	Type AccountType `gorm:"not null;type:varchar(20)" json:"type"`

	// Password stores the hashed password (only for EMAIL type accounts)
	// Not exposed in JSON responses
	Password string `gorm:"size:255" json:"-"`

	// IsPrimary indicates if this is the user's primary account
	IsPrimary bool `gorm:"default:false" json:"isPrimary"`
}

// TableName specifies the database table name
func (Account) TableName() string {
	return "accounts"
}

// BeforeCreate ensures only one primary account per user
func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.IsPrimary {
		// Set all other accounts of this user to non-primary
		tx.Model(&Account{}).
			Where("user_id = ? AND is_primary = ?", a.UserID, true).
			Update("is_primary", false)
	}
	return nil
}
