package repository

import (
	"qris-latency-optimizer/domain/entity"
)

type TransactionRepository interface {
	Create(tx *entity.Transaction) error
	FindByID(id string) (*entity.Transaction, error)
	UpdateStatus(id string, status string) (int64, error)
}
