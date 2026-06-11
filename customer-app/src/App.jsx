import { useState } from 'react';
import QRScanner from './components/QRScanner';
import PaymentForm from './components/PaymentForm';
import TransactionStatus from './components/TransactionStatus';
import { 
  scanQR, 
  extractMerchantFromQRIS, 
  extractAmountFromQRIS 
} from './services/api';
import './App.css';

function App() {
  const [screen, setScreen] = useState('scanner'); // scanner | form | status
  const [scannedQR, setScannedQR] = useState('');
  const [extractedData, setExtractedData] = useState({
    merchantId: '',
    amount: 0
  });
  const [transactionId, setTransactionId] = useState('');
  const [loading, setLoading] = useState(false);

  const handleQRScanned = (qrData) => {
    console.log('QR Scanned:', qrData);
    
    // DIUBAH: Extract merchant ID dan amount dari QR payload
    const merchantId = extractMerchantFromQRIS(qrData);
    const amount = extractAmountFromQRIS(qrData);
    
    console.log('Extracted Merchant ID:', merchantId);
    console.log('Extracted Amount:', amount);
    
    setScannedQR(qrData);
    setExtractedData({
      merchantId: merchantId || '',
      amount: amount || 0
    });
    setScreen('form');
  };

  const handlePaymentSubmit = async (formData) => {
    setLoading(true);
    try {
      // DIUBAH: Gunakan merchant ID yang di-extract dari QR
      const response = await scanQR(
        scannedQR,
        extractedData.merchantId, // Dari QR payload bukan form
        extractedData.amount // Dari QR payload bukan form
      );

      if (response.data && response.data.transaction_id) {
        setTransactionId(response.data.transaction_id);
        setScreen('status');
      } else {
        alert('Failed to create transaction');
      }
    } catch (error) {
      console.error('Error:', error);
      alert('Error: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  const handleBack = () => {
    setScannedQR('');
    setExtractedData({ merchantId: '', amount: 0 });
    setTransactionId('');
    setScreen('scanner');
  };

  return (
    <div className="app">
      {screen === 'scanner' && (
        <QRScanner onScan={handleQRScanned} isScanning={true} />
      )}

      {screen === 'form' && (
        <PaymentForm
          onSubmit={handlePaymentSubmit}
          isLoading={loading}
          scannedQR={scannedQR}
          extractedData={extractedData} // BARU: Pass extracted data
        />
      )}

      {screen === 'status' && (
        <TransactionStatus
          transactionId={transactionId}
          onBack={handleBack}
        />
      )}
    </div>
  );
}

export default App;