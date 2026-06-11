package redis

import (
	"encoding/json"
	"fmt"

	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/repository/postgres"
)

func merchantCacheKey(qrID string) string {
	return "merchant:" + qrID
}

type MerchantCache struct{}

func NewMerchantCache() MerchantCache {
	return MerchantCache{}
}

func (MerchantCache) GetMerchant(qrID string) (*entity.Merchant, bool) {
	return GetMerchant(qrID)
}

func (MerchantCache) CacheMerchant(merchant entity.Merchant) {
	CacheMerchant(merchant)
}

type MerchantPrefetcher struct{}

func NewMerchantPrefetcher() MerchantPrefetcher {
	return MerchantPrefetcher{}
}

func (MerchantPrefetcher) PrefetchRelatedMerchants(currentQRID string) {
	PrefetchRelatedMerchants(currentQRID)
}

// PrefetchMerchant fetches one merchant from DB and stores it in Redis.
func PrefetchMerchant(qrID string) {
	if !RedisAvailable || qrID == "" {
		return
	}

	cacheKey := merchantCacheKey(qrID)
	exists, err := Exists(cacheKey)
	if err == nil && exists {
		return
	}

	var merchant entity.Merchant
	if err := postgres.DB.Where("qr_id = ? AND is_active = ?", qrID, true).First(&merchant).Error; err != nil {
		return
	}

	data, err := json.Marshal(merchant)
	if err != nil {
		return
	}

	_ = Set(cacheKey, string(data), TTLMerchant)
}

// PrefetchRelatedMerchants speculatively prefetches other active merchants.
func PrefetchRelatedMerchants(currentQRID string) {
	if !RedisAvailable || currentQRID == "" {
		return
	}

	var merchants []entity.Merchant
	if err := postgres.DB.
		Where("is_active = ? AND qr_id != ?", true, currentQRID).
		Limit(5).
		Find(&merchants).Error; err != nil {
		return
	}

	for _, merchant := range merchants {
		cacheKey := merchantCacheKey(merchant.QRID)
		exists, err := Exists(cacheKey)
		if err == nil && exists {
			continue
		}

		data, err := json.Marshal(merchant)
		if err != nil {
			continue
		}

		_ = Set(cacheKey, string(data), TTLMerchant/2)
	}
}

// WarmUpCache pre-loads all active merchants into Redis at startup.
func WarmUpCache() {
	if !RedisAvailable {
		return
	}

	var merchants []entity.Merchant
	if err := postgres.DB.Where("is_active = ?", true).Find(&merchants).Error; err != nil {
		return
	}

	for _, merchant := range merchants {
		data, err := json.Marshal(merchant)
		if err != nil {
			continue
		}

		_ = Set(merchantCacheKey(merchant.QRID), string(data), TTLMerchant)
	}
}

func GetMerchant(qrID string) (*entity.Merchant, bool) {
	if !RedisAvailable || qrID == "" {
		return nil, false
	}

	cachedData, err := Get(merchantCacheKey(qrID))
	if err != nil || cachedData == "" {
		return nil, false
	}

	var merchant entity.Merchant
	if err := json.Unmarshal([]byte(cachedData), &merchant); err != nil {
		_ = Delete(merchantCacheKey(qrID))
		return nil, false
	}

	return &merchant, true
}

func CacheMerchant(merchant entity.Merchant) {
	if !RedisAvailable || merchant.QRID == "" {
		return
	}

	data, err := json.Marshal(merchant)
	if err != nil {
		return
	}

	_ = Set(merchantCacheKey(merchant.QRID), string(data), TTLMerchant)
}

func DeleteMerchant(qrID string) error {
	if qrID == "" {
		return nil
	}

	return Delete(fmt.Sprintf("merchant:%s", qrID))
}
