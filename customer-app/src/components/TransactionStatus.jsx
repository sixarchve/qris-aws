import { useEffect, useState } from 'react';
import { getTransactionStatus, confirmPayment } from '../services/api';
import '../styles/TransactionStatus.css';

export default function TransactionStatus({ transactionId, onBack }) {
  const [transaction, setTransaction] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [confirming, setConfirming] = useState(false);

  useEffect(() => {
    fetchStatus();
    // Poll status setiap 2 detik
    const interval = setInterval(fetchStatus, 2000);
    return () => clearInterval(interval);
  }, [transactionId]);

  const fetchStatus = async () => {
    try {
      const response = await getTransactionStatus(transactionId);
      setTransaction(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to fetch transaction status');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleConfirm = async () => {
    setConfirming(true);
    try {
      const response = await confirmPayment(transactionId);
      setTransaction(response.data);
      alert('Payment confirmed successfully!');
    } catch (err) {
      setError('Failed to confirm payment');
      console.error(err);
    } finally {
      setConfirming(false);
    }
  };

  if (loading) {
    return <div className="status-loading">Loading transaction...</div>;
  }

  if (error) {
    return (
      <div className="status-error">
        <p>{error}</p>
        <button onClick={onBack}>Back</button>
      </div>
    );
  }

  return (
    <div className="status-container">
      <div className="status-card">
        <h2>Transaction Status</h2>

        <div className="status-info">
          <div className="info-row">
            <span className="label">Transaction ID:</span>
            <span className="value">{transaction?.transaction_id}</span>
          </div>

          <div className="info-row">
            <span className="label">Amount:</span>
            <span className="value">
              IDR {transaction?.amount?.toLocaleString('id-ID')}
            </span>
          </div>

          <div className="info-row">
            <span className="label">Status:</span>
            <span
              className={`status-badge ${transaction?.status?.toLowerCase()}`}
            >
              {transaction?.status}
            </span>
          </div>

          <div className="info-row">
            <span className="label">From Cache:</span>
            <span className="value">
              {transaction?.cached_from ? 'Yes (Fast!)' : 'No (Database)'}
            </span>
          </div>

          <div className="info-row">
            <span className="label">Created:</span>
            <span className="value">
              {new Date(transaction?.created_at).toLocaleString()}
            </span>
          </div>
        </div>

        <div className="status-actions">
          {transaction?.status === 'PENDING' && (
            <button
              onClick={handleConfirm}
              disabled={confirming}
              className="confirm-button"
            >
              {confirming ? 'Confirming...' : 'Confirm Payment'}
            </button>
          )}

          {transaction?.status === 'SUCCESS' && (
            <div className="success-message">
              Payment confirmed successfully!
            </div>
          )}

          <button onClick={onBack} className="back-button">
            Scan Another QR
          </button>
        </div>
      </div>
    </div>
  );
}
