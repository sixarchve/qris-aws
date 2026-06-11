import { useState, useEffect } from 'react';
import '../styles/PaymentForm.css';
import { extractMerchantFromQRIS, extractAmountFromQRIS } from '../services/api';

export default function PaymentForm({ onSubmit, isLoading, scannedQR }) {
  const [formData, setFormData] = useState({
    qrPayload: scannedQR || '',
    merchantId: '',
    amount: '',
  });

  // BARU: Extract data dari QR saat component mount atau scannedQR berubah
  useEffect(() => {
    if (scannedQR) {
      const extractedMerchantId = extractMerchantFromQRIS(scannedQR);
      const extractedAmount = extractAmountFromQRIS(scannedQR);

      console.log('Extracted Merchant ID:', extractedMerchantId);
      console.log('Extracted Amount:', extractedAmount);

      setFormData((prev) => ({
        ...prev,
        qrPayload: scannedQR,
        merchantId: extractedMerchantId || '',
        amount: extractedAmount || '',
      }));
    }
  }, [scannedQR]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!formData.qrPayload || !formData.merchantId || !formData.amount) {
      alert('Please fill all fields');
      return;
    }
    onSubmit(formData);
  };

  return (
    <div className="payment-form-container">
      <div className="form-card">
        <h2>Payment Details</h2>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>QR Payload *</label>
            <textarea
              name="qrPayload"
              value={formData.qrPayload}
              onChange={handleChange}
              placeholder="QR code data will appear here"
              rows="3"
              disabled
              className="qr-payload-field"
            />
            <small>Scanned automatically from QR code</small>
          </div>

          <div className="form-group">
            <label>Merchant ID *</label>
            <input
              type="text"
              name="merchantId"
              value={formData.merchantId}
              onChange={handleChange}
              placeholder="Extracted from QR code"
              disabled
            />
            <small>Automatically extracted from QR code</small>
          </div>

          <div className="form-group">
            <label>Amount (IDR) *</label>
            <input
              type="number"
              name="amount"
              value={formData.amount}
              onChange={handleChange}
              placeholder="Extracted from QR code"
              min="1000"
              step="1000"
              disabled
            />
            <small>Automatically extracted from QR code</small>
          </div>

          <button
            type="submit"
            disabled={isLoading || !formData.merchantId || !formData.amount}
            className="submit-button"
          >
            {isLoading ? 'Processing...' : 'Confirm Payment'}
          </button>
        </form>
      </div>
    </div>
  );
}
