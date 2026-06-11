import { useEffect, useRef, useState } from 'react';

function buildWebSocketURL(merchantId) {
  if (import.meta.env.VITE_WS_URL) {
    const configuredURL = new URL(import.meta.env.VITE_WS_URL);
    configuredURL.searchParams.set('merchant_id', merchantId);
    return configuredURL.toString();
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsURL = new URL(`${protocol}//${window.location.host}/ws`);
  wsURL.searchParams.set('merchant_id', merchantId);
  return wsURL.toString();
}

export default function NotificationListener({
  onNotificationReceived,
  merchantId,
  isActive = true,
}) {
  const [isConnected, setIsConnected] = useState(false);
  const [retryCount, setRetryCount] = useState(0);
  const callbackRef = useRef(onNotificationReceived);
  const maxRetries = 10;

  useEffect(() => {
    callbackRef.current = onNotificationReceived;
  }, [onNotificationReceived]);

  useEffect(() => {
    if (!merchantId || !isActive) {
      console.warn('WebSocket skipped: merchant ID missing or listener inactive');
      queueMicrotask(() => {
        setIsConnected(false);
        setRetryCount(0);
      });
      return undefined;
    }

    let socket;
    let reconnectTimer;
    let shouldReconnect = true;
    let attempt = 0;

    const connectWebSocket = () => {
      const wsURL = buildWebSocketURL(merchantId);

      console.log(`Connecting to WebSocket: ${wsURL}`);
      socket = new WebSocket(wsURL);

      socket.onopen = () => {
        console.log('WebSocket connected');
        attempt = 0;
        setIsConnected(true);
        setRetryCount(0);
      };

      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('WebSocket message received:', message);

          if (message.type === 'transaction_notification') {
            callbackRef.current?.(message);
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      socket.onerror = (error) => {
        console.error('WebSocket error:', error);
        setIsConnected(false);
      };

      socket.onclose = () => {
        console.log('WebSocket disconnected');
        setIsConnected(false);

        if (!shouldReconnect) {
          return;
        }

        if (attempt < maxRetries) {
          const nextAttempt = attempt + 1;
          const delayMs = Math.min(1000 * Math.pow(2, attempt), 30000);
          attempt = nextAttempt;
          setRetryCount(nextAttempt);
          console.log(`Reconnecting WebSocket in ${delayMs}ms (attempt ${nextAttempt}/${maxRetries})`);

          reconnectTimer = setTimeout(connectWebSocket, delayMs);
        } else {
          console.error('Max WebSocket retries reached');
        }
      };
    };

    connectWebSocket();

    return () => {
      shouldReconnect = false;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      if (socket) {
        socket.onclose = null;
        socket.close(1000, 'listener cleanup');
      }
    };
  }, [merchantId, isActive]);

  return (
    <div style={{
      fontSize: '12px',
      position: 'fixed',
      bottom: '10px',
      left: '10px',
      zIndex: 9998,
    }}>
      <span style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '8px',
        background: isConnected ? '#ecfdf5' : '#fef2f2',
        padding: '8px 12px',
        borderRadius: '8px',
        border: `1px solid ${isConnected ? '#d1fae5' : '#fecaca'}`,
        color: isConnected ? '#047857' : '#991b1b',
        fontWeight: '500',
      }}>
        WebSocket:
        <span>
          {isConnected ? 'Connected' : 'Disconnected'}
        </span>
        {!isConnected && retryCount > 0 && retryCount <= maxRetries && (
          <span style={{ fontSize: '10px', marginLeft: '4px' }}>
            (retry {retryCount}/{maxRetries})
          </span>
        )}
      </span>
    </div>
  );
}
