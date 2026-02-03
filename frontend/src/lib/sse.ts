export type SSEEventType = 'emoji' | 'archive';

export interface SSEEmojiData {
  Type: 1;
  Data: {
    card_id: number;
    emojis: Array<{ value: string; count: number }>;
  };
}

export interface SSEArchiveData {
  Type: 2;
  Data: number;
}

export type SSEData = SSEEmojiData | SSEArchiveData;

export function createSSEConnection(
  onEmoji: (cardId: number, emojis: Array<{ value: string; count: number }>) => void,
  onArchive: (cardId: number) => void
): EventSource {
  // Connect directly to backend for SSE (Next.js rewrites may not handle SSE properly)
  const sseUrl = typeof window !== 'undefined' && window.location.hostname === 'localhost'
    ? 'http://localhost:8080/api/sse'
    : '/api/sse';

  console.log('[SSE] Connecting to:', sseUrl);
  const sse = new EventSource(sseUrl);

  sse.onopen = () => {
    console.log('[SSE] Connection opened');
  };

  sse.onmessage = (event) => {
    console.log('[SSE] onmessage received:', event.data);
  };

  sse.addEventListener('connected', (event) => {
    console.log('[SSE] connected event:', event.data);
  });

  sse.addEventListener('heartbeat', (event) => {
    console.log('[SSE] heartbeat event:', event.data);
  });

  sse.addEventListener('emoji', (event) => {
    console.log('[SSE] emoji event received:', event.data);
    try {
      const data: SSEData = JSON.parse(event.data);
      console.log('[SSE] emoji event parsed:', data);
      if (data.Type === 1) {
        console.log('[SSE] Calling onEmoji for card:', data.Data.card_id);
        onEmoji(data.Data.card_id, data.Data.emojis);
      } else if (data.Type === 2) {
        console.log('[SSE] Calling onArchive for card:', data.Data);
        onArchive(data.Data);
      }
    } catch (e) {
      console.error('[SSE] Failed to parse emoji event:', e);
    }
  });

  sse.onerror = (error) => {
    console.error('[SSE] Error:', error);
    console.log('[SSE] ReadyState:', sse.readyState);
  };

  return sse;
}
