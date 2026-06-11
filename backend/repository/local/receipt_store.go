package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"qris-latency-optimizer/domain/entity"
	"time"
)

type ReceiptStore struct {
	dir string
}

type receiptFile struct {
	ReceiptID     string    `json:"receipt_id"`
	TransactionID string    `json:"transaction_id"`
	MerchantID    string    `json:"merchant_id"`
	MerchantName  string    `json:"merchant_name"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	IssuedAt      time.Time `json:"issued_at"`
}

func NewReceiptStore(dir string) *ReceiptStore {
	return &ReceiptStore{dir: dir}
}

func (s *ReceiptStore) SaveReceipt(tx entity.Transaction) (string, error) {
	if s == nil || s.dir == "" {
		return "", nil
	}

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return "", fmt.Errorf("create receipt directory: %w", err)
	}

	receipt := receiptFile{
		ReceiptID:     fmt.Sprintf("receipt_%s", tx.ID.String()),
		TransactionID: tx.ID.String(),
		MerchantID:    tx.MerchantID.String(),
		MerchantName:  tx.Merchant.MerchantName,
		Amount:        tx.Amount,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
		IssuedAt:      time.Now(),
	}

	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode receipt: %w", err)
	}

	path := filepath.Join(s.dir, receipt.ReceiptID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write receipt: %w", err)
	}

	return path, nil
}
