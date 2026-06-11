import { useCallback, useEffect, useRef, useState } from "react";
import { QRCodeCanvas } from "qrcode.react";
import NotificationListener from "./components/NotificationListener";
import "./App.css";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api";

export default function App() {
  const [merchants, setMerchants] = useState([]);
  const [selectedMerchantId, setSelectedMerchantId] = useState("");
  const [selectedMerchantInfo, setSelectedMerchantInfo] = useState(null);
  const [payload, setPayload] = useState("");
  const [loading, setLoading] = useState(true);
  const [qrError, setQrError] = useState("");
  const [inputAmount, setInputAmount] = useState(1000);
  const [submittedAmount, setSubmittedAmount] = useState(1000);

  // Notification state
  const [showNotification, setShowNotification] = useState(false);
  const [currentNotification, setCurrentNotification] = useState(null);
  const hideNotificationTimerRef = useRef(null);

  useEffect(() => {
    return () => {
      if (hideNotificationTimerRef.current) {
        clearTimeout(hideNotificationTimerRef.current);
      }
    };
  }, []);

  // Load merchant list
  useEffect(() => {
    const fetchMerchants = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/merchants`);
        const data = await response.json();

        const normalizedMerchants = data.merchants.map(m => ({
          id: m.ID,
          qr_id: m.QRID,
          merchant_name: m.MerchantName,
          is_active: m.IsActive,
          created_at: m.CreatedAt
        }));

        setMerchants(normalizedMerchants);
        if (normalizedMerchants.length > 0) {
          setSelectedMerchantId(normalizedMerchants[0].id);
          setSelectedMerchantInfo(normalizedMerchants[0]);
        }
      } catch (err) {
        console.error("Failed to fetch merchants", err);
      }
    };
    fetchMerchants();
  }, []);

  // Generate QRIS
  useEffect(() => {
    if (!selectedMerchantId || !submittedAmount || submittedAmount <= 0) {
      queueMicrotask(() => setPayload(""));
      queueMicrotask(() => setQrError(""));
      return;
    }

    queueMicrotask(() => setLoading(true));
    queueMicrotask(() => setQrError(""));
    fetch(
      `${API_BASE_URL}/qris?merchant_id=${selectedMerchantId}&amount=${submittedAmount}`
    )
      .then(async (res) => {
        const data = await res.json();
        if (!res.ok) {
          throw new Error(data.error || `QRIS request failed with status ${res.status}`);
        }
        return data;
      })
      .then((data) => {
        if (!data.qris_payload) {
          throw new Error("QRIS payload missing from response");
        }
        setPayload(data.qris_payload);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to fetch QRIS payload", err);
        setPayload("");
        setQrError(err.message || "Failed to generate QR");
        setLoading(false);
      });
  }, [selectedMerchantId, submittedAmount]);

  const handleMerchantChange = (e) => {
    const merchantId = e.target.value;
    setSelectedMerchantId(merchantId);

    const selected = merchants.find((m) => m.id === merchantId);
    setSelectedMerchantInfo(selected);

    setInputAmount(1000);
    setSubmittedAmount(1000);
  };

  // Handler untuk notification dari WebSocket
  const handleNotificationReceived = useCallback((notification) => {
    console.log("🔔 Notification received:", notification);

    setCurrentNotification(notification);
    setShowNotification(true);

    // Auto-hide setelah 5 detik
    if (hideNotificationTimerRef.current) {
      clearTimeout(hideNotificationTimerRef.current);
    }

    hideNotificationTimerRef.current = setTimeout(() => {
      setShowNotification(false);
    }, 8000);

  }, []);

  const formatRupiah = new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
  });

  return (
    <div style={styles.page}>
      {/* WebSocket Listener dengan merchant_id */}
      <NotificationListener
        onNotificationReceived={handleNotificationReceived}
        merchantId={selectedMerchantId}
        isActive={true}
      />

      {/* Notification Banner */}
      {showNotification && currentNotification && (
        <div style={styles.notificationBanner}>
          <div style={styles.notificationContent}>
            <span style={styles.notificationBell}>🔔</span>
            <div style={styles.notificationText}>
              <p style={styles.notificationTitle}>New Transaction!</p>
              <p style={styles.notificationDetails}>
                Merchant: <strong>{currentNotification.merchant_name}</strong>
              </p>
              <p style={styles.notificationDetails}>
                ID: <code style={styles.notificationCode}>{currentNotification.merchant_id}</code>
              </p>
              <p style={styles.notificationDetails}>
                Amount: <strong>{formatRupiah.format(currentNotification.amount || 0)}</strong>
              </p>
              <p style={{ ...styles.notificationDetails, fontSize: '11px', color: 'rgba(255,255,255,0.7)' }}>
                {new Date().toLocaleTimeString()}
              </p>
            </div>
          </div>
        </div>
      )}

      <div style={styles.card}>
        <div style={styles.header}>
          <div>
            <p style={styles.qrisText}>QRIS PAYMENT</p>
            <h1 style={styles.title}>Capstone Pay</h1>
          </div>
          <div style={styles.liveBadge}>
            <span style={styles.dot}></span> Live
          </div>
        </div>

        <div style={styles.merchantSelector}>
          <label style={{ color: "black", fontWeight: "bold", fontSize: "12px" }}>
            SELECT MERCHANT
          </label>
          <select
            value={selectedMerchantId}
            onChange={handleMerchantChange}
            style={styles.selectDropdown}
          >
            <option value="">-- Select Merchant --</option>
            {merchants.map((m) => (
              <option key={m.id} value={m.id}>
                {m.merchant_name}
              </option>
            ))}
          </select>
        </div>

        <div style={styles.merchant}>
          <h2 style={{ color: "black" }}>
            {selectedMerchantInfo?.merchant_name || "Select Merchant"}
          </h2>
          <p style={{ color: "black" }}>
            {selectedMerchantInfo?.qr_id || ""}
          </p>
        </div>

        <div style={styles.amountBox}>
          <p style={{ color: "black", fontSize: "12px", fontWeight: "bold" }}>
            ENTER AMOUNT (IDR)
          </p>
          <div style={{ display: "flex", gap: "8px", marginTop: "8px" }}>
            <input
              type="number"
              value={inputAmount}
              onChange={(e) =>
                setInputAmount(e.target.value === "" ? "" : Number(e.target.value))
              }
              style={styles.amountInput}
              min="1"
            />
            <button
              onClick={() => setSubmittedAmount(inputAmount)}
              style={styles.generateButton}
              disabled={!selectedMerchantId}
            >
              Generate
            </button>
          </div>
        </div>

        <div style={styles.qrWrapper}>
          {loading ? (
            <p>Generating QR...</p>
          ) : qrError ? (
            <p style={{ color: "#b91c1c", textAlign: "center" }}>{qrError}</p>
          ) : payload ? (
            <>
              <QRCodeCanvas value={payload} size={220} />
              <div style={styles.scanLine}></div>
            </>
          ) : (
            <p style={{ color: "#999" }}>Select merchant and amount to generate QR</p>
          )}
        </div>

        <p style={styles.instruction}>
          Scan this QR to pay using any QRIS-supported app
        </p>

        <div style={styles.logoRow}>
          {["QRIS Supported", "All Banks", "E-Wallet"].map((b) => (
            <span key={b} style={styles.logo}>
              {b}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}

const styles = {
  page: {
    minHeight: "100vh",
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    background: "#f4f4f4",
  },
  card: {
    width: "100%",
    maxWidth: "360px",
    borderRadius: "24px",
    background: "white",
    overflow: "hidden",
    boxShadow: "0 15px 40px rgba(0,0,0,0.12)",
  },
  header: {
    background: "linear-gradient(135deg, #ef4444, #b91c1c)",
    color: "white",
    padding: "20px",
    display: "flex",
    justifyContent: "space-between",
  },
  qrisText: {
    fontSize: "11px",
    letterSpacing: "2px",
  },
  title: {
    marginTop: "6px",
    fontSize: "36px",
    lineHeight: "1.2",
    maxWidth: "200px",
  },
  liveBadge: {
    background: "rgba(255,255,255,0.2)",
    padding: "6px 12px",
    borderRadius: "20px",
    fontSize: "12px",
    alignSelf: "flex-start",
  },
  dot: {
    height: "6px",
    width: "6px",
    background: "white",
    borderRadius: "50%",
    display: "inline-block",
    marginRight: "6px",
  },
  merchantSelector: {
    margin: "20px",
    padding: "12px",
    background: "#f9f9f9",
    borderRadius: "8px",
  },
  selectDropdown: {
    width: "100%",
    marginTop: "8px",
    padding: "10px",
    fontSize: "14px",
    border: "1px solid #ddd",
    borderRadius: "6px",
    backgroundColor: "white",
    color: "black",
    cursor: "pointer",
    outline: "none",
  },
  merchant: {
    textAlign: "center",
    padding: "20px",
  },
  amountBox: {
    margin: "0 20px",
    padding: "16px",
    background: "#fce7e7",
    borderRadius: "16px",
    textAlign: "center",
  },
  amountInput: {
    flex: 1,
    fontSize: "20px",
    fontWeight: "bold",
    textAlign: "center",
    width: "100%",
    border: "none",
    background: "white",
    borderRadius: "8px",
    color: "black",
    outline: "none",
    padding: "8px",
  },
  generateButton: {
    padding: "8px 16px",
    background: "#ef4444",
    color: "white",
    border: "none",
    borderRadius: "8px",
    fontWeight: "bold",
    cursor: "pointer",
    boxShadow: "0 4px 6px rgba(239, 68, 68, 0.3)",
  },
  qrWrapper: {
    margin: "20px",
    padding: "16px",
    background: "#fff",
    borderRadius: "16px",
    border: "1px solid #eee",
    position: "relative",
    textAlign: "center",
  },
  scanLine: {
    position: "absolute",
    width: "100%",
    height: "3px",
    background: "rgba(255,0,0,0.4)",
    top: "50%",
    animation: "scan 2s infinite",
  },
  instruction: {
    textAlign: "center",
    fontSize: "12px",
    color: "#666",
  },
  logoRow: {
    display: "flex",
    justifyContent: "center",
    gap: "10px",
    margin: "20px",
    flexWrap: "wrap",
  },
  logo: {
    fontSize: "10px",
    background: "#eee",
    padding: "6px 10px",
    borderRadius: "12px",
  },
  notificationBanner: {
    position: "fixed",
    top: "20px",
    right: "20px",
    background: "linear-gradient(135deg, #10b981, #059669)",
    color: "white",
    padding: "16px 24px",
    borderRadius: "12px",
    boxShadow: "0 10px 30px rgba(16, 185, 129, 0.3)",
    zIndex: 9999,
    maxWidth: "400px",
    animation: "slideIn 0.3s ease-out",
  },
  notificationContent: {
    display: "flex",
    gap: "16px",
    alignItems: "flex-start",
  },
  notificationBell: {
    fontSize: "24px",
  },
  notificationText: {
    flex: 1,
  },
  notificationTitle: {
    margin: "0 0 8px 0",
    fontSize: "16px",
    fontWeight: "bold",
  },
  notificationDetails: {
    margin: "4px 0",
    fontSize: "13px",
  },
  notificationCode: {
    background: "rgba(255, 255, 255, 0.2)",
    padding: "2px 6px",
    borderRadius: "4px",
    fontFamily: "monospace",
    fontSize: "12px",
  },
};
