import { useEffect, useRef, useState } from 'react';
import QrScanner from 'qr-scanner';
import '../styles/QRScanner.css';

export default function QRScanner({ onScan, isScanning }) {
  const videoRef = useRef(null);
  const [error, setError] = useState(null);
  const [torchSupported, setTorchSupported] = useState(false);
  const [torchActive, setTorchActive] = useState(false);
  const qrScannerRef = useRef(null);

  useEffect(() => {
    if (!videoRef.current || !isScanning) return;

    const qrScanner = new QrScanner(
      videoRef.current,
      (result) => {
        // QR code detected
        onScan(result.data);
        qrScanner.stop();
      },
      {
        onDecodeError: (error) => {
          // Scanning error - ignore
        },
        maxScansPerSecond: 5,
      }
    );

    qrScannerRef.current = qrScanner;

    // Check if torch is supported
    qrScanner.hasFlash().then((supported) => {
      setTorchSupported(supported);
    });

    qrScanner.start().catch((err) => {
      console.error('Error starting scanner:', err);
      setError('Unable to access camera. Please check permissions.');
    });

    return () => {
      qrScanner.destroy();
    };
  }, [isScanning, onScan]);

  const toggleTorch = async () => {
    if (qrScannerRef.current) {
      try {
        await qrScannerRef.current.toggleFlash();
        setTorchActive(!torchActive);
      } catch (err) {
        console.error('Error toggling torch:', err);
      }
    }
  };

  return (
    <div className="qr-scanner-container">
      <div className="scanner-header">
        <h2>Scan QR Code</h2>
        <p>Point camera at QR code</p>
      </div>

      {error && <div className="error-message">{error}</div>}

      <div className="video-wrapper">
        <video ref={videoRef}></video>
        <div className="scanner-overlay">
          <div className="corner corner-top-left"></div>
          <div className="corner corner-top-right"></div>
          <div className="corner corner-bottom-left"></div>
          <div className="corner corner-bottom-right"></div>
        </div>
      </div>

      {torchSupported && (
        <button
          className={`torch-button ${torchActive ? 'active' : ''}`}
          onClick={toggleTorch}
        >
          {torchActive ? 'Torch On' : 'Torch Off'}
        </button>
      )}

      <div className="scanner-info">
        <p>Scanning in progress...</p>
      </div>
    </div>
  );
}
