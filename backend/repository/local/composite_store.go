package local

import (
	"log"
	"qris-latency-optimizer/domain/entity"
	"qris-latency-optimizer/usecase"
)

type CompositeReceiptStore struct {
	stores []usecase.ReceiptStore
}

func NewCompositeReceiptStore(stores ...usecase.ReceiptStore) *CompositeReceiptStore {
	return &CompositeReceiptStore{stores: stores}
}

func (c *CompositeReceiptStore) SaveReceipt(tx entity.Transaction) (string, error) {
	var primaryPath string
	var lastErr error
	for _, store := range c.stores {
		if store == nil {
			continue
		}
		path, err := store.SaveReceipt(tx)
		if err != nil {
			log.Printf("CompositeReceiptStore: failed to save to sub-store: %v", err)
			lastErr = err
		} else if primaryPath == "" {
			// Save the first successful store's returned path as the primary one
			primaryPath = path
		}
	}
	return primaryPath, lastErr
}
