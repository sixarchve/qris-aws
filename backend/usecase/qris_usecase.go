package usecase

import (
	"errors"
	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/domain/repository"
	"strconv"

	"github.com/google/uuid"
)

type QRISUsecase interface {
	GenerateQRIS(merchantIDStr string, amountStr string) (string, *entity.Merchant, int, error)
}

type qrisUsecase struct {
	repo             repository.MerchantRepository
	merchantCache    MerchantCache
	merchantPrefetch MerchantPrefetcher
	qrisCodec        QRISCodec
}

func NewQRISUsecase(
	repo repository.MerchantRepository,
	merchantCache MerchantCache,
	merchantPrefetch MerchantPrefetcher,
	qrisCodec QRISCodec,
) QRISUsecase {
	return &qrisUsecase{
		repo:             repo,
		merchantCache:    merchantCache,
		merchantPrefetch: merchantPrefetch,
		qrisCodec:        qrisCodec,
	}
}

func (u *qrisUsecase) GenerateQRIS(merchantIDStr string, amountStr string) (string, *entity.Merchant, int, error) {
	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 {
		return "", nil, 0, errors.New("invalid amount")
	}

	if merchantIDStr == "" {
		return "", nil, 0, errors.New("merchant_id is required")
	}

	merchantUUID, err := uuid.Parse(merchantIDStr)
	if err != nil {
		return "", nil, 0, errors.New("invalid merchant_id")
	}

	merchant, err := u.repo.FindByID(merchantUUID)
	if err != nil {
		return "", nil, 0, errors.New("merchant not found")
	}

	// Cache operations
	u.merchantCache.CacheMerchant(*merchant)
	go u.merchantPrefetch.PrefetchRelatedMerchants(merchant.QRID)

	payload, err := u.qrisCodec.GeneratePayload(amount, merchant.MerchantName, merchant.QRID)
	if err != nil {
		return "", nil, 0, err
	}

	return payload, merchant, amount, nil
}
