package models

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a user login session
type Session struct {
	BaseModel

	// SessionToken is the unique token for this session (stored in cookie)
	SessionToken string `gorm:"uniqueIndex;not null;size:255" json:"sessionToken"`

	// UserID is the foreign key to the user
	UserID uuid.UUID `gorm:"not null;index" json:"userId"`

	// User relationship (many sessions belong to one user)
	User User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`

	// ExpiresAt is when this session becomes invalid
	ExpiresAt time.Time `gorm:"not null;index" json:"expiresAt"`

	// IPAddress stores the IP address where the session was created
	IPAddress string `gorm:"size:45" json:"ipAddress"`

	// UserAgent stores the browser/user agent string
	UserAgent string `gorm:"size:500" json:"userAgent"`

	// RememberMe indicates if this session should persist longer (remember me)
	RememberMe bool `gorm:"not null;default:false" json:"rememberMe"`
}

// TableName specifies the database table name
func (Session) TableName() string {
	return "sessions"
}

// IsValid checks if the session is still valid (not expired)
func (s *Session) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}
