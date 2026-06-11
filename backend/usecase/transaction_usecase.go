package usecase

import (
	"errors"
	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/domain/repository"
	"time"

	"github.com/google/uuid"
)

// TransactionUsecase is the public interface used by HTTP handlers.
type TransactionUsecase interface {
	ScanQR(req entity.ScanQRRequest) (*entity.TransactionResponse, error)
	ConfirmPaymentAsync(transactionIDStr string) error
	GetTransactionStatus(transactionIDStr string) (*entity.TransactionResponse, error)
}

// PaymentWorkerUsecase is used only by the payment worker goroutine to
// finalise a queued payment. It is intentionally separate from
// TransactionUsecase so it is never exposed as an HTTP route.
type PaymentWorkerUsecase interface {
	ConfirmPayment(transactionIDStr string) (*entity.TransactionResponse, error)
}

type transactionUsecase struct {
	txRepo                repository.TransactionRepository
	merchantRepo          repository.MerchantRepository
	txCache               TransactionCache
	merchantCache         MerchantCache
	paymentPublisher      PaymentPublisher
	notificationPublisher NotificationPublisher
	qrisCodec             QRISCodec
}

func NewTransactionUsecase(
	txRepo repository.TransactionRepository,
	merchantRepo repository.MerchantRepository,
	txCache TransactionCache,
	merchantCache MerchantCache,
	paymentPublisher PaymentPublisher,
	notificationPublisher NotificationPublisher,
	qrisCodec QRISCodec,
) *transactionUsecase {
	return &transactionUsecase{
		txRepo:                txRepo,
		merchantRepo:          merchantRepo,
		txCache:               txCache,
		merchantCache:         merchantCache,
		paymentPublisher:      paymentPublisher,
		notificationPublisher: notificationPublisher,
		qrisCodec:             qrisCodec,
	}
}

func (u *transactionUsecase) ScanQR(req entity.ScanQRRequest) (*entity.TransactionResponse, error) {
	var merchant *entity.Merchant

	merchantUUID, err := uuid.Parse(req.MerchantID)
	if err == nil {
		merchant, err = u.merchantRepo.FindByID(merchantUUID)
		if err != nil {
			return nil, errors.New("merchant not found")
		}
	} else {
		// cache lookup
		if cached, ok := u.merchantCache.GetMerchant(req.MerchantID); ok {
			merchant = cached
		} else {
			merchant, err = u.merchantRepo.FindByQRID(req.MerchantID)
			if err != nil {
				return nil, errors.New("merchant not found")
			}
			u.merchantCache.CacheMerchant(*merchant)
		}
	}

	qrMerchantID, qrAmount, err := u.qrisCodec.ParsePayload(req.QRPayload)
	if err != nil {
		return nil, err
	}
	if qrMerchantID != merchant.QRID {
		return nil, errors.New("qr payload merchant does not match merchant id")
	}
	if float64(qrAmount) != req.Amount {
		return nil, errors.New("qr payload amount does not match amount")
	}

	tx := entity.Transaction{
		ID:         uuid.New(),
		MerchantID: merchant.ID,
		Amount:     req.Amount,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
	}

	if err := u.txRepo.Create(&tx); err != nil {
		return nil, errors.New("failed to create transaction")
	}

	u.txCache.CacheTransaction(tx)

	return &entity.TransactionResponse{
		TransactionID: tx.ID.String(),
		MerchantID:    merchant.ID.String(),
		Amount:        tx.Amount,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
		CachedFrom:    false,
	}, nil
}

func (u *transactionUsecase) ConfirmPaymentAsync(transactionIDStr string) error {
	if _, err := uuid.Parse(transactionIDStr); err != nil {
		return errors.New("invalid transaction id")
	}

	err := u.paymentPublisher.PublishPaymentConfirmation(transactionIDStr)
	if err != nil {
		return errors.New("failed to queue transaction: " + err.Error())
	}
	return nil
}

// ConfirmPayment updates the transaction to SUCCESS, invalidates the cache,
// and publishes a merchant notification. It is called by the payment worker.
func (u *transactionUsecase) ConfirmPayment(transactionIDStr string) (*entity.TransactionResponse, error) {
	if _, err := uuid.Parse(transactionIDStr); err != nil {
		return nil, errors.New("invalid transaction id")
	}

	rows, err := u.txRepo.UpdateStatus(transactionIDStr, "SUCCESS")
	if err != nil {
		return nil, errors.New("failed to confirm payment")
	}
	if rows == 0 {
		return nil, errors.New("transaction not found")
	}

	u.txCache.DeleteTransaction(transactionIDStr)

	tx, err := u.txRepo.FindByID(transactionIDStr)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	go func() {
		_ = u.notificationPublisher.PublishNotification(
			tx.ID.String(),
			tx.MerchantID.String(),
			tx.Merchant.MerchantName,
			tx.Amount,
		)
	}()

	return &entity.TransactionResponse{
		TransactionID: tx.ID.String(),
		MerchantID:    tx.MerchantID.String(),
		Amount:        tx.Amount,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
	}, nil
}

func (u *transactionUsecase) GetTransactionStatus(transactionIDStr string) (*entity.TransactionResponse, error) {
	if _, err := uuid.Parse(transactionIDStr); err != nil {
		return nil, errors.New("invalid transaction id")
	}

	// Cache check
	if tx, ok := u.txCache.GetTransaction(transactionIDStr); ok {
		return &entity.TransactionResponse{
			TransactionID: tx.ID.String(),
			MerchantID:    tx.MerchantID.String(),
			Amount:        tx.Amount,
			Status:        tx.Status,
			CreatedAt:     tx.CreatedAt,
			CachedFrom:    true,
		}, nil
	}

	tx, err := u.txRepo.FindByID(transactionIDStr)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	u.txCache.CacheTransaction(*tx)

	return &entity.TransactionResponse{
		TransactionID: tx.ID.String(),
		MerchantID:    tx.MerchantID.String(),
		Amount:        tx.Amount,
		Status:        tx.Status,
		CreatedAt:     tx.CreatedAt,
		CachedFrom:    false,
	}, nil
}
