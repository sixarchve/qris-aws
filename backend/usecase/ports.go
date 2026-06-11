package usecase

import "qris-latency-optimizer/domain/entity"

type MerchantCache interface {
	GetMerchant(qrID string) (*entity.Merchant, bool)
	CacheMerchant(merchant entity.Merchant)
}

type TransactionCache interface {
	GetTransaction(id string) (*entity.Transaction, bool)
	CacheTransaction(tx entity.Transaction)
	DeleteTransaction(id string)
}

type MerchantPrefetcher interface {
	PrefetchRelatedMerchants(currentQRID string)
}

type PaymentPublisher interface {
	PublishPaymentConfirmation(transactionID string) error
}

type NotificationPublisher interface {
	PublishNotification(txID, merchantID, merchantName string, amount float64) error
}

type QRISCodec interface {
	GeneratePayload(amount int, merchantName string, qrID string) (string, error)
	ParsePayload(payload string) (string, int, error)
}
