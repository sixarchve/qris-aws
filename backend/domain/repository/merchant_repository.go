package repository

import (
	"qris-latency-optimizer/domain/entity"

	"github.com/google/uuid"
)

type MerchantRepository interface {
	FindByID(id uuid.UUID) (*entity.Merchant, error)
	FindByQRID(qrID string) (*entity.Merchant, error)
	FindAllActive() ([]entity.Merchant, error)
}
