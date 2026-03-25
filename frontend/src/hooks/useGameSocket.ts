import { useEffect, useState, useCallback, useRef } from 'react';

export interface SSEEvent {
  type: string;
  payload: any;
}

export interface UseGameSocketResult {
  connected: boolean;
  lastEvent: SSEEvent | null;
  error: string | null;
}

export function useGameSocket(
  gameId: string | null,
  onEvent?: (event: SSEEvent) => void
): UseGameSocketResult {
  const [connected, setConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState<SSEEvent | null>(null);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);

  const connect = useCallback(() => {
    // Get token from localStorage (stored by Activity Hub SDK)
    const token = localStorage.getItem('token');
    if (!gameId || !token) return;

    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // Use absolute URL for SSE to avoid base path issues
    const url = `/api/apps/bulls-and-cows/proxy/api/game/${gameId}/stream?token=${encodeURIComponent(token)}`;

    console.log('[SSE] Connecting to:', url);

    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      console.log('[SSE] Connected');
      setConnected(true);
      setError(null);
      reconnectAttemptsRef.current = 0;
    };

    eventSource.onmessage = (event) => {
      try {
        const data: SSEEvent = JSON.parse(event.data);
        console.log('[SSE] Received event:', data);
        setLastEvent(data);
        if (onEvent) {
          onEvent(data);
        }
      } catch (err) {
        console.error('[SSE] Failed to parse event:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.error('[SSE] Connection error:', err);
      setConnected(false);
      setError('Connection lost. Reconnecting...');
      eventSource.close();

      // Exponential backoff for reconnection
      const backoff = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
      reconnectAttemptsRef.current++;

      console.log(`[SSE] Reconnecting in ${backoff}ms (attempt ${reconnectAttemptsRef.current})`);

      reconnectTimeoutRef.current = setTimeout(() => {
        connect();
      }, backoff);
    };
  }, [gameId, onEvent]);

  useEffect(() => {
    if (gameId) {
      connect();
    }

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
    };
  }, [gameId, connect]);

  return { connected, lastEvent, error };
}
