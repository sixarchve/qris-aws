package local

import (
	"errors"
	"qris-latency-optimizer/domain/entity"
	"testing"

	"github.com/google/uuid"
)

type mockSubStore struct {
	saveCalled bool
	returnPath string
	returnErr  error
	receivedTx entity.Transaction
}

func (m *mockSubStore) SaveReceipt(tx entity.Transaction) (string, error) {
	m.saveCalled = true
	m.receivedTx = tx
	return m.returnPath, m.returnErr
}

func TestCompositeReceiptStore(t *testing.T) {
	tx := entity.Transaction{
		ID: uuid.New(),
	}

	t.Run("calls all sub-stores and returns first path", func(t *testing.T) {
		s1 := &mockSubStore{returnPath: "s3://path"}
		s2 := &mockSubStore{returnPath: "local/path"}
		composite := NewCompositeReceiptStore(s1, s2)

		path, err := composite.SaveReceipt(tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !s1.saveCalled || !s2.saveCalled {
			t.Fatal("expected all sub-stores to be called")
		}

		if path != "s3://path" {
			t.Fatalf("expected primary path to be 's3://path', got '%s'", path)
		}
	})

	t.Run("skips nil stores and handles errors", func(t *testing.T) {
		s1 := &mockSubStore{returnErr: errors.New("s3 upload failed")}
		s2 := &mockSubStore{returnPath: "local/path"}
		composite := NewCompositeReceiptStore(s1, nil, s2)

		path, err := composite.SaveReceipt(tx)
		if err == nil {
			t.Fatal("expected an error to be returned")
		}

		if !s1.saveCalled || !s2.saveCalled {
			t.Fatal("expected both sub-stores to be called")
		}

		if path != "local/path" {
			t.Fatalf("expected primary path to be the first successful one 'local/path', got '%s'", path)
		}
	})
}
