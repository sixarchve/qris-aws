package entity

import (
	"time"

	"github.com/google/uuid"
)

type Merchant struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	QRID         string    `gorm:"column:qr_id;type:varchar(50);uniqueIndex;not null"`
	MerchantName string    `gorm:"column:merchant_name;type:varchar(100);not null"`
	IsActive     bool      `gorm:"column:is_active;default:true"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`

	Transactions []Transaction `gorm:"foreignKey:MerchantID"`
}
