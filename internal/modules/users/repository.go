package users

import (
	"context"

	"fiberest/internal/database"
	"fiberest/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ponytail: single Repository per module is a DDD-lite simplification; no aggregates/events.
// Upgrade path: split into UserRepository + AccountRepository or introduce domain layer when needed.

// Repository abstracts persistence for the users module.
// Matches current service needs exactly; not speculative.
type Repository interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByEmailExcludingID(ctx context.Context, email, id string) (bool, error)
	Create(ctx context.Context, user *models.User) error
	CreateWithAccount(ctx context.Context, user *models.User, account *models.Account) error
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindAccount(ctx context.Context, userID uuid.UUID, accountType models.AccountType) (*models.Account, error)
	CreateAccount(ctx context.Context, account *models.Account) error
	SaveAccount(ctx context.Context, account *models.Account) error
	SaveUser(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int, role, search string) ([]models.User, int64, error)
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

func (r *gormRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := r.getDB(ctx).Model(&models.User{}).Select("count(*) > 0").Where("email = ?", email).Find(&exists).Error; err != nil {
		return false, err
	}
	return exists, nil
}

func (r *gormRepository) ExistsByEmailExcludingID(ctx context.Context, email, id string) (bool, error) {
	var exists bool
	// Use count(*) > 0 pattern consistent with ExistsByEmail; lean query check.
	var count int64
	if err := r.getDB(ctx).Model(&models.User{}).Where("email = ? AND id != ?", email, id).Count(&count).Error; err != nil {
		return false, err
	}
	exists = count > 0
	return exists, nil
}

func (r *gormRepository) Create(ctx context.Context, user *models.User) error {
	return r.getDB(ctx).Create(user).Error
}

func (r *gormRepository) CreateWithAccount(ctx context.Context, user *models.User, account *models.Account) error {
	tx := r.getDB(ctx).Begin()
	defer func() {
		if rec := recover(); rec != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return err
	}
	account.UserID = user.ID
	if err := tx.Create(account).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *gormRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.getDB(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
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

func (r *gormRepository) CreateAccount(ctx context.Context, account *models.Account) error {
	return r.getDB(ctx).Create(account).Error
}

func (r *gormRepository) SaveAccount(ctx context.Context, account *models.Account) error {
	return r.getDB(ctx).Save(account).Error
}

func (r *gormRepository) SaveUser(ctx context.Context, user *models.User) error {
	return r.getDB(ctx).Save(user).Error
}

func (r *gormRepository) Delete(ctx context.Context, id string) error {
	return r.getDB(ctx).Delete(&models.User{}, "id = ?", id).Error
}

func (r *gormRepository) List(ctx context.Context, limit, offset int, role, search string) ([]models.User, int64, error) {
	query := r.getDB(ctx).Model(&models.User{})
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []models.User
	if err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
