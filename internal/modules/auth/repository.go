package auth

import (
	"context"

	"fiberest/internal/database"
	"fiberest/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ponytail: single Repository per module is a DDD-lite simplification; no aggregates/events.
// Upgrade path: split into UserRepository/SessionRepository/AccountRepository or add domain layer.

// Repository abstracts persistence for the auth module.
// Matches current service needs exactly; not speculative.
type Repository interface {
	CountAdmins(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, user *models.User) error
	CreateAccount(ctx context.Context, account *models.Account) error
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	FindAccount(ctx context.Context, userID uuid.UUID, accountType models.AccountType) (*models.Account, error)
	UpdateAccountPassword(ctx context.Context, account *models.Account, hashedPassword string) error
	CreateSession(ctx context.Context, session *models.Session) error
	FindSessionByToken(ctx context.Context, token string) (*models.Session, error)
	FindSessionByID(ctx context.Context, id uuid.UUID) (*models.Session, error)
	DeleteSessionByToken(ctx context.Context, token string) error
	DeleteSessionByID(ctx context.Context, id uuid.UUID) error
	DeleteSessionsByUserIDExceptToken(ctx context.Context, userID uuid.UUID, keepToken string) error
	FindSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]models.Session, int64, error)
	UpdateSession(ctx context.Context, session *models.Session, fields map[string]any) error
}

type gormRepository struct {
	db *database.DatabaseService
}

// NewRepository provides a GORM-backed Repository.
func NewRepository(db *database.DatabaseService) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) getDB(ctx context.Context) *gorm.DB {
	return r.db.GetDB().WithContext(ctx)
}

func (r *gormRepository) CountAdmins(ctx context.Context) (int64, error) {
	var count int64
	if err := r.getDB(ctx).Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *gormRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.getDB(ctx).Create(user).Error
}

func (r *gormRepository) CreateAccount(ctx context.Context, account *models.Account) error {
	return r.getDB(ctx).Create(account).Error
}

func (r *gormRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.getDB(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) FindAccount(ctx context.Context, userID uuid.UUID, accountType models.AccountType) (*models.Account, error) {
	var account models.Account
	if err := r.getDB(ctx).Where("user_id = ? AND type = ?", userID, accountType).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *gormRepository) UpdateAccountPassword(ctx context.Context, account *models.Account, hashedPassword string) error {
	return r.getDB(ctx).Model(account).Update("password", hashedPassword).Error
}

func (r *gormRepository) CreateSession(ctx context.Context, session *models.Session) error {
	return r.getDB(ctx).Create(session).Error
}

func (r *gormRepository) FindSessionByToken(ctx context.Context, token string) (*models.Session, error) {
	var session models.Session
	if err := r.getDB(ctx).Preload("User").Where("session_token = ?", token).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *gormRepository) FindSessionByID(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	var session models.Session
	if err := r.getDB(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *gormRepository) DeleteSessionByToken(ctx context.Context, token string) error {
	return r.getDB(ctx).Where("session_token = ?", token).Delete(&models.Session{}).Error
}

func (r *gormRepository) DeleteSessionByID(ctx context.Context, id uuid.UUID) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&models.Session{}).Error
}

func (r *gormRepository) DeleteSessionsByUserIDExceptToken(ctx context.Context, userID uuid.UUID, keepToken string) error {
	return r.getDB(ctx).
		Where("user_id = ? AND session_token <> ?", userID, keepToken).
		Delete(&models.Session{}).Error
}

func (r *gormRepository) FindSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]models.Session, int64, error) {
	var total int64
	if err := r.getDB(ctx).Model(&models.Session{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var sessions []models.Session
	if err := r.getDB(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}

func (r *gormRepository) UpdateSession(ctx context.Context, session *models.Session, fields map[string]any) error {
	return r.getDB(ctx).Model(session).Updates(fields).Error
}
