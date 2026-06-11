package entity

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MerchantID uuid.UUID `gorm:"column:merchant_id;type:uuid;index"`
	Amount     float64   `gorm:"type:decimal(15,2);not null"`
	Status     string    `gorm:"type:varchar(20);default:'PENDING'"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`

	Merchant Merchant `gorm:"foreignKey:MerchantID"`
}

// request entity model saat scan QR
type ScanQRRequest struct {
	QRPayload  string  `json:"qr_payload" binding:"required"`
	MerchantID string  `json:"merchant_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
}

// response entity GetTransactionStatus
type TransactionResponse struct {
	TransactionID string    `json:"transaction_id"`
	MerchantID    string    `json:"merchant_id"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	CachedFrom    bool      `json:"cached_from,omitempty"`
}
