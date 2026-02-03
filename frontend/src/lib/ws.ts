import { EmojiData } from './api';

export interface WSEmojiData {
  Type: 1;
  Data: {
    card_id: number;
    emojis: Array<{ value: string; count: number }>;
  };
}

export interface WSArchiveData {
  Type: 2;
  Data: number;
}

export type WSData = WSEmojiData | WSArchiveData;

type EmojiCallback = (cardId: number, emojis: EmojiData[]) => void;
type ArchiveCallback = (cardId: number) => void;
type CursorCallback = (clientId: string, cardId: number, x: number, y: number) => void;
type CursorLeaveCallback = (clientId: string) => void;

interface Subscriber {
  id: string;
  onEmoji: EmojiCallback;
  onArchive: ArchiveCallback;
  onCursor?: CursorCallback;
  onCursorLeave?: CursorLeaveCallback;
}

// Singleton WebSocket manager
class WSManager {
  private ws: WebSocket | null = null;
  private clientId: string | null = null;
  private subscribers: Map<string, Subscriber> = new Map();
  private reconnectTimer: NodeJS.Timeout | null = null;
  private isConnecting = false;

  private getWsUrl(): string {
    const wsProtocol = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsHost = typeof window !== 'undefined' && window.location.hostname === 'localhost'
      ? 'localhost:8080'
      : window.location.host;
    return `${wsProtocol}//${wsHost}/api/ws`;
  }

  private connect() {
    if (typeof window === 'undefined') return;
    if (this.isConnecting) return;
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    this.isConnecting = true;
    const wsUrl = this.getWsUrl();
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log('[WS] Connected');
      this.isConnecting = false;
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        // Handle connection message
        if (data.type === 'connected') {
          this.clientId = data.clientId;
          return;
        }

        // Handle cursor updates
        if (data.type === 'cursor') {
          this.subscribers.forEach(sub => sub.onCursor?.(data.clientId, data.cardId, data.x, data.y));
          return;
        }

        // Handle cursor leave
        if (data.type === 'cursor_leave') {
          this.subscribers.forEach(sub => sub.onCursorLeave?.(data.clientId));
          return;
        }

        // Handle emoji/archive events
        const wsData = data as WSData;
        if (wsData.Type === 1) {
          this.subscribers.forEach(sub => sub.onEmoji(wsData.Data.card_id, wsData.Data.emojis));
        } else if (wsData.Type === 2) {
          this.subscribers.forEach(sub => sub.onArchive(wsData.Data));
        }
      } catch (e) {
        console.error('[WS] Failed to parse message:', e);
      }
    };

    this.ws.onerror = (error) => {
      console.error('[WS] Error:', error);
      this.isConnecting = false;
    };

    this.ws.onclose = () => {
      console.log('[WS] Disconnected');
      this.isConnecting = false;
      this.ws = null;

      // Reconnect if there are still subscribers
      if (this.subscribers.size > 0) {
        this.scheduleReconnect();
      }
    };
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.subscribers.size > 0) {
        this.connect();
      }
    }, 2000);
  }

  subscribe(subscriber: Omit<Subscriber, 'id'>): () => void {
    const id = Math.random().toString(36).substring(2, 15);
    this.subscribers.set(id, { ...subscriber, id });

    // Connect if not already connected
    if (!this.ws || this.ws.readyState === WebSocket.CLOSED) {
      this.connect();
    }

    // Return unsubscribe function
    return () => {
      this.subscribers.delete(id);

      // Don't close connection immediately - wait a bit in case of StrictMode remount
      setTimeout(() => {
        if (this.subscribers.size === 0) {
          if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
          }
          if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.close();
            this.ws = null;
          }
        }
      }, 100);
    };
  }

  sendCursor(cardId: number, x: number, y: number) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'cursor',
        cardId,
        x,
        y,
      }));
    }
  }

  getClientId(): string | null {
    return this.clientId;
  }
}

// Export singleton instance
export const wsManager = new WSManager();
