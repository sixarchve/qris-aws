package postgres

import (
	"qris-latency-optimizer/domain/entity"

	"gorm.io/gorm"
)

type transactionRepo struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *transactionRepo {
	return &transactionRepo{db: db}
}

func (r *transactionRepo) Create(tx *entity.Transaction) error {
	return r.db.Create(tx).Error
}

func (r *transactionRepo) FindByID(id string) (*entity.Transaction, error) {
	var tx entity.Transaction
	err := r.db.Preload("Merchant").First(&tx, "id = ?", id).Error
	return &tx, err
}

func (r *transactionRepo) UpdateStatus(id string, status string) (int64, error) {
	result := r.db.Model(&entity.Transaction{}).Where("id = ?", id).Update("status", status)
	return result.RowsAffected, result.Error
}
