package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"qris-latency-optimizer/domain/entity"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3ReceiptStore struct {
	s3Client   *s3.Client
	bucketName string
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

func NewS3ReceiptStore(cfg aws.Config, bucketName string) *S3ReceiptStore {
	return &S3ReceiptStore{
		s3Client:   s3.NewFromConfig(cfg),
		bucketName: bucketName,
	}
}

func (s *S3ReceiptStore) SaveReceipt(tx entity.Transaction) (string, error) {
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
		return "", fmt.Errorf("encode S3 receipt: %w", err)
	}

	key := fmt.Sprintf("receipts/receipt_%s.json", tx.ID.String())

	_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return "", fmt.Errorf("upload receipt to S3: %w", err)
	}

	// Construct the S3 URL
	s3URL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucketName, key)
	return s3URL, nil
}
