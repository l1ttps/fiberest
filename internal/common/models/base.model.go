package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel provides common fields for all database models.
// It uses UUID as the primary key instead of auto-incrementing integers
// for better distributed system support and security.
type BaseModel struct {
	// ID is the primary key using UUID v4 format
	// GORM tag ensures it's stored as uuid type in PostgreSQL with primary key constraint
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// CreatedAt tracks when the record was created
	// Automatically set by GORM on create
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"createdAt"`

	// UpdatedAt tracks when the record was last modified
	// Automatically updated by GORM on save/update
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updatedAt"`
}

// BeforeCreate hook ensures ID is generated before inserting a new record.
// This provides UUID generation at the application level as a fallback
// in case the database default is not available.
func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}
