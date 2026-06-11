import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// BARU: Extract merchant ID dari QRIS payload
export const extractMerchantFromQRIS = (payload) => {
  if (!payload) return null;
  
  let i = 0;
  while (i < payload.length - 4) {
    const tag = payload.substring(i, i + 2);
    const len = parseInt(payload.substring(i + 2, i + 4), 10);
    
    if (Number.isNaN(len)) break;
    
    const value = payload.substring(i + 4, i + 4 + len);
    
    // Tag 26 = Merchant Information yang berisi ID merchant
    if (tag === "26") {
      // Parse merchant info yang berformat TLV juga
      let j = 0;
      while (j < value.length - 4) {
        const innerTag = value.substring(j, j + 2);
        const innerLen = parseInt(value.substring(j + 2, j + 4), 10);
        
        if (Number.isNaN(innerLen)) break;
        
        const innerValue = value.substring(j + 4, j + 4 + innerLen);
        
        // Tag 01 dalam merchant info = merchant ID
        if (innerTag === "01") {
          return innerValue;
        }
        
        j += 4 + innerLen;
      }
    }
    
    i += 4 + len;
  }
  
  return null;
};

// BARU: Extract amount dari QRIS payload
export const extractAmountFromQRIS = (payload) => {
  if (!payload) return 0;
  
  let i = 0;
  while (i < payload.length - 4) {
    const tag = payload.substring(i, i + 2);
    const len = parseInt(payload.substring(i + 2, i + 4), 10);
    
    if (Number.isNaN(len)) break;
    
    const value = payload.substring(i + 4, i + 4 + len);
    
    // Tag 54 = Amount
    if (tag === "54") {
      return parseInt(value, 10);
    }
    
    i += 4 + len;
  }
  
  return 0;
};

// Scan QR - Create Transaction
export const scanQR = async (qrPayload, merchantId, amount) => {
  try {
    const response = await api.post('/transactions/scan', {
      qr_payload: qrPayload,
      merchant_id: merchantId,
      amount: parseFloat(amount),
    });
    return response.data;
  } catch (error) {
    console.error('Error scanning QR:', error);
    throw error;
  }
};

// Get Transaction Status
export const getTransactionStatus = async (transactionId) => {
  try {
    const response = await api.get(`/transactions/${transactionId}`);
    return response.data;
  } catch (error) {
    console.error('Error getting transaction status:', error);
    throw error;
  }
};

// Confirm Payment
export const confirmPayment = async (transactionId) => {
  try {
    const response = await api.post(`/transactions/${transactionId}/confirm`);
    return response.data;
  } catch (error) {
    console.error('Error confirming payment:', error);
    throw error;
  }
};

export default api;