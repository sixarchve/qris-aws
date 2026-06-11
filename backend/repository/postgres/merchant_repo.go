package postgres

import (
	"qris-latency-optimizer/domain/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type merchantRepo struct {
	db *gorm.DB
}

func NewMerchantRepository(db *gorm.DB) *merchantRepo {
	return &merchantRepo{db: db}
}

func (r *merchantRepo) FindByID(id uuid.UUID) (*entity.Merchant, error) {
	var merchant entity.Merchant
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&merchant).Error
	return &merchant, err
}

func (r *merchantRepo) FindByQRID(qrID string) (*entity.Merchant, error) {
	var merchant entity.Merchant
	err := r.db.Where("qr_id = ? AND is_active = ?", qrID, true).First(&merchant).Error
	return &merchant, err
}

func (r *merchantRepo) FindAllActive() ([]entity.Merchant, error) {
	var merchants []entity.Merchant
	err := r.db.Where("is_active = ?", true).Find(&merchants).Error
	return merchants, err
}
