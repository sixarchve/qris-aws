package usecase

import (
	"errors"
	"testing"
	"time"

	"qris-latency-optimizer/domain/entity"

	"github.com/google/uuid"
)

type fakeMerchantRepo struct {
	byID   map[uuid.UUID]*entity.Merchant
	byQRID map[string]*entity.Merchant
}

func (r *fakeMerchantRepo) FindByID(id uuid.UUID) (*entity.Merchant, error) {
	if merchant, ok := r.byID[id]; ok {
		return merchant, nil
	}
	return nil, errors.New("not found")
}

func (r *fakeMerchantRepo) FindByQRID(qrID string) (*entity.Merchant, error) {
	if merchant, ok := r.byQRID[qrID]; ok {
		return merchant, nil
	}
	return nil, errors.New("not found")
}

func (r *fakeMerchantRepo) FindAllActive() ([]entity.Merchant, error) {
	return nil, nil
}

type fakeTransactionRepo struct {
	byID       map[string]*entity.Transaction
	created    *entity.Transaction
	updateRows int64
	updateErr  error
}

func (r *fakeTransactionRepo) Create(tx *entity.Transaction) error {
	copy := *tx
	r.created = &copy
	r.byID[tx.ID.String()] = &copy
	return nil
}

func (r *fakeTransactionRepo) FindByID(id string) (*entity.Transaction, error) {
	if tx, ok := r.byID[id]; ok {
		return tx, nil
	}
	return nil, errors.New("not found")
}

func (r *fakeTransactionRepo) UpdateStatus(id string, status string) (int64, error) {
	if r.updateErr != nil {
		return 0, r.updateErr
	}
	if tx, ok := r.byID[id]; ok {
		tx.Status = status
		return r.updateRows, nil
	}
	return 0, nil
}

type fakeMerchantCache struct {
	merchants map[string]*entity.Merchant
	cached    []entity.Merchant
}

func (c *fakeMerchantCache) GetMerchant(qrID string) (*entity.Merchant, bool) {
	merchant, ok := c.merchants[qrID]
	return merchant, ok
}

func (c *fakeMerchantCache) CacheMerchant(merchant entity.Merchant) {
	c.cached = append(c.cached, merchant)
}

type fakeTransactionCache struct {
	transactions map[string]*entity.Transaction
	cached       []entity.Transaction
	deleted      []string
}

func (c *fakeTransactionCache) GetTransaction(id string) (*entity.Transaction, bool) {
	tx, ok := c.transactions[id]
	return tx, ok
}

func (c *fakeTransactionCache) CacheTransaction(tx entity.Transaction) {
	c.cached = append(c.cached, tx)
}

func (c *fakeTransactionCache) DeleteTransaction(id string) {
	c.deleted = append(c.deleted, id)
}

type fakePaymentPublisher struct {
	published []string
	err       error
}

func (p *fakePaymentPublisher) PublishPaymentConfirmation(transactionID string) error {
	p.published = append(p.published, transactionID)
	return p.err
}

type fakeNotificationPublisher struct {
	published chan string
}

func (p *fakeNotificationPublisher) PublishNotification(txID, merchantID, merchantName string, amount float64) error {
	p.published <- txID
	return nil
}

type fakeQRISCodec struct {
	merchantID string
	amount     int
	parseErr   error
}

func (c fakeQRISCodec) GeneratePayload(amount int, merchantName string, qrID string) (string, error) {
	return "payload", nil
}

func (c fakeQRISCodec) ParsePayload(payload string) (string, int, error) {
	return c.merchantID, c.amount, c.parseErr
}

func newTransactionUsecaseFixture(merchant *entity.Merchant) (*transactionUsecase, *fakeTransactionRepo, *fakeTransactionCache, *fakeMerchantCache, *fakePaymentPublisher, *fakeNotificationPublisher) {
	txRepo := &fakeTransactionRepo{byID: map[string]*entity.Transaction{}, updateRows: 1}
	merchantRepo := &fakeMerchantRepo{
		byID:   map[uuid.UUID]*entity.Merchant{merchant.ID: merchant},
		byQRID: map[string]*entity.Merchant{merchant.QRID: merchant},
	}
	txCache := &fakeTransactionCache{transactions: map[string]*entity.Transaction{}}
	merchantCache := &fakeMerchantCache{merchants: map[string]*entity.Merchant{}}
	paymentPublisher := &fakePaymentPublisher{}
	notificationPublisher := &fakeNotificationPublisher{published: make(chan string, 1)}

	u := NewTransactionUsecase(
		txRepo,
		merchantRepo,
		txCache,
		merchantCache,
		paymentPublisher,
		notificationPublisher,
		fakeQRISCodec{merchantID: merchant.QRID, amount: 15000},
	)

	return u, txRepo, txCache, merchantCache, paymentPublisher, notificationPublisher
}

func TestTransactionUsecaseScanQRByQRIDCacheMiss(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	u, txRepo, txCache, merchantCache, _, _ := newTransactionUsecaseFixture(merchant)

	resp, err := u.ScanQR(entity.ScanQRRequest{
		QRPayload:  "payload",
		MerchantID: merchant.QRID,
		Amount:     15000,
	})
	if err != nil {
		t.Fatalf("ScanQR returned error: %v", err)
	}
	if resp.MerchantID != merchant.ID.String() || resp.Status != "PENDING" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if txRepo.created == nil {
		t.Fatal("expected transaction to be created")
	}
	if len(txCache.cached) != 1 {
		t.Fatalf("expected transaction cache write, got %d", len(txCache.cached))
	}
	if len(merchantCache.cached) != 1 {
		t.Fatalf("expected merchant cache write after repo lookup, got %d", len(merchantCache.cached))
	}
}

func TestTransactionUsecaseScanQRByUUID(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	u, _, _, merchantCache, _, _ := newTransactionUsecaseFixture(merchant)

	resp, err := u.ScanQR(entity.ScanQRRequest{
		QRPayload:  "payload",
		MerchantID: merchant.ID.String(),
		Amount:     15000,
	})
	if err != nil {
		t.Fatalf("ScanQR returned error: %v", err)
	}
	if resp.MerchantID != merchant.ID.String() {
		t.Fatalf("expected merchant %s, got %s", merchant.ID, resp.MerchantID)
	}
	if len(merchantCache.cached) != 0 {
		t.Fatalf("did not expect QRID merchant cache write for UUID lookup, got %d", len(merchantCache.cached))
	}
}

func TestTransactionUsecaseScanQRValidationErrors(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	u, _, _, _, _, _ := newTransactionUsecaseFixture(merchant)

	u.qrisCodec = fakeQRISCodec{parseErr: errors.New("invalid qris payload")}
	if _, err := u.ScanQR(entity.ScanQRRequest{QRPayload: "bad", MerchantID: merchant.QRID, Amount: 15000}); err == nil {
		t.Fatal("expected invalid QR payload error")
	}

	u.qrisCodec = fakeQRISCodec{merchantID: "OTHER", amount: 15000}
	if _, err := u.ScanQR(entity.ScanQRRequest{QRPayload: "payload", MerchantID: merchant.QRID, Amount: 15000}); err == nil {
		t.Fatal("expected merchant mismatch error")
	}

	u.qrisCodec = fakeQRISCodec{merchantID: merchant.QRID, amount: 20000}
	if _, err := u.ScanQR(entity.ScanQRRequest{QRPayload: "payload", MerchantID: merchant.QRID, Amount: 15000}); err == nil {
		t.Fatal("expected amount mismatch error")
	}
}

func TestTransactionUsecaseConfirmPaymentAsyncPublishesEvent(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	u, _, _, _, paymentPublisher, _ := newTransactionUsecaseFixture(merchant)
	txID := uuid.New().String()

	if err := u.ConfirmPaymentAsync(txID); err != nil {
		t.Fatalf("ConfirmPaymentAsync returned error: %v", err)
	}
	if len(paymentPublisher.published) != 1 || paymentPublisher.published[0] != txID {
		t.Fatalf("unexpected published events: %+v", paymentPublisher.published)
	}
}

func TestTransactionUsecaseConfirmPaymentUpdatesCacheAndNotification(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	u, txRepo, txCache, _, _, notificationPublisher := newTransactionUsecaseFixture(merchant)
	txID := uuid.New()
	txRepo.byID[txID.String()] = &entity.Transaction{
		ID:         txID,
		MerchantID: merchant.ID,
		Merchant:   *merchant,
		Amount:     15000,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
	}

	resp, err := u.ConfirmPayment(txID.String())
	if err != nil {
		t.Fatalf("ConfirmPayment returned error: %v", err)
	}
	if resp.Status != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", resp.Status)
	}
	if len(txCache.deleted) != 1 || txCache.deleted[0] != txID.String() {
		t.Fatalf("expected cache delete for %s, got %+v", txID, txCache.deleted)
	}

	select {
	case publishedID := <-notificationPublisher.published:
		if publishedID != txID.String() {
			t.Fatalf("expected notification for %s, got %s", txID, publishedID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected notification publish")
	}
}

func TestTransactionUsecaseGetTransactionStatusCacheHitAndMiss(t *testing.T) {
	merchant := &entity.Merchant{ID: uuid.New(), QRID: "TEST001", MerchantName: "Kantin", IsActive: true}
	u, txRepo, txCache, _, _, _ := newTransactionUsecaseFixture(merchant)
	cachedID := uuid.New()
	txCache.transactions[cachedID.String()] = &entity.Transaction{
		ID:         cachedID,
		MerchantID: merchant.ID,
		Amount:     15000,
		Status:     "PENDING",
		CreatedAt:  time.Now(),
	}

	resp, err := u.GetTransactionStatus(cachedID.String())
	if err != nil {
		t.Fatalf("GetTransactionStatus cache hit returned error: %v", err)
	}
	if !resp.CachedFrom {
		t.Fatal("expected cached response")
	}

	dbID := uuid.New()
	txRepo.byID[dbID.String()] = &entity.Transaction{
		ID:         dbID,
		MerchantID: merchant.ID,
		Amount:     20000,
		Status:     "SUCCESS",
		CreatedAt:  time.Now(),
	}

	resp, err = u.GetTransactionStatus(dbID.String())
	if err != nil {
		t.Fatalf("GetTransactionStatus DB fallback returned error: %v", err)
	}
	if resp.CachedFrom {
		t.Fatal("did not expect DB fallback response to be marked cached")
	}
	if len(txCache.cached) != 1 || txCache.cached[0].ID != dbID {
		t.Fatalf("expected DB fallback cache write for %s, got %+v", dbID, txCache.cached)
	}
}
