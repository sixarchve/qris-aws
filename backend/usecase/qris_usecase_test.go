package usecase

import (
	"testing"
	"time"

	"qris-latency-optimizer/domain/entity"

	"github.com/google/uuid"
)

type fakePrefetcher struct {
	qrIDs chan string
}

func (p *fakePrefetcher) PrefetchRelatedMerchants(currentQRID string) {
	p.qrIDs <- currentQRID
}

func TestQRISUsecaseGenerateQRIS(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	merchantRepo := &fakeMerchantRepo{
		byID:   map[uuid.UUID]*entity.Merchant{merchant.ID: merchant},
		byQRID: map[string]*entity.Merchant{},
	}
	merchantCache := &fakeMerchantCache{merchants: map[string]*entity.Merchant{}}
	prefetcher := &fakePrefetcher{qrIDs: make(chan string, 1)}

	u := NewQRISUsecase(merchantRepo, merchantCache, prefetcher, fakeQRISCodec{})

	payload, gotMerchant, amount, err := u.GenerateQRIS(merchant.ID.String(), "15000")
	if err != nil {
		t.Fatalf("GenerateQRIS returned error: %v", err)
	}
	if payload != "payload" || gotMerchant.ID != merchant.ID || amount != 15000 {
		t.Fatalf("unexpected result: payload=%q merchant=%+v amount=%d", payload, gotMerchant, amount)
	}
	if len(merchantCache.cached) != 1 || merchantCache.cached[0].ID != merchant.ID {
		t.Fatalf("expected merchant cache write, got %+v", merchantCache.cached)
	}
	select {
	case qrID := <-prefetcher.qrIDs:
		if qrID != merchant.QRID {
			t.Fatalf("expected prefetch for %s, got %s", merchant.QRID, qrID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected merchant prefetch")
	}
}

func TestQRISUsecaseGenerateQRISValidatesInput(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	u := NewQRISUsecase(
		&fakeMerchantRepo{byID: map[uuid.UUID]*entity.Merchant{merchant.ID: merchant}},
		&fakeMerchantCache{merchants: map[string]*entity.Merchant{}},
		&fakePrefetcher{qrIDs: make(chan string, 1)},
		fakeQRISCodec{},
	)

	tests := []struct {
		name       string
		merchantID string
		amount     string
	}{
		{name: "invalid amount", merchantID: merchant.ID.String(), amount: "abc"},
		{name: "zero amount", merchantID: merchant.ID.String(), amount: "0"},
		{name: "missing merchant id", merchantID: "", amount: "15000"},
		{name: "invalid merchant id", merchantID: "not-a-uuid", amount: "15000"},
		{name: "merchant not found", merchantID: uuid.New().String(), amount: "15000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, _, err := u.GenerateQRIS(tt.merchantID, tt.amount); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
