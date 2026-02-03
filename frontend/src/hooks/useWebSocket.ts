import { useState, useEffect, useCallback } from 'react';
import { EmojiData } from '@/lib/api';
import { wsManager } from '@/lib/ws';

const CURSOR_COLORS = [
  '#ef4444', '#f97316', '#eab308', '#22c55e', '#14b8a6',
  '#3b82f6', '#8b5cf6', '#ec4899', '#f43f5e', '#06b6d4',
];

function getCursorColor(clientId: string): string {
  let hash = 0;
  for (let i = 0; i < clientId.length; i++) {
    hash = clientId.charCodeAt(i) + ((hash << 5) - hash);
  }
  return CURSOR_COLORS[Math.abs(hash) % CURSOR_COLORS.length];
}

export interface CursorPosition {
  cardId: number;
  x: number;
  y: number;
  color: string;
}

export interface RemoteCursor {
  clientId: string;
  x: number;
  y: number;
  color: string;
}

export function useWebSocket() {
  const [emojiUpdates, setEmojiUpdates] = useState<Record<number, EmojiData[]>>({});
  const [archiveUpdates, setArchiveUpdates] = useState<Record<number, boolean>>({});
  const [cursors, setCursors] = useState<Record<string, CursorPosition>>({});

  useEffect(() => {
    const unsubscribe = wsManager.subscribe({
      onEmoji: (cardId, emojis) => {
        setEmojiUpdates((prev) => ({ ...prev, [cardId]: emojis }));
      },
      onArchive: (cardId) => {
        setArchiveUpdates((prev) => ({ ...prev, [cardId]: true }));
      },
      onCursor: (clientId, cardId, x, y) => {
        setCursors((prev) => ({
          ...prev,
          [clientId]: { cardId, x, y, color: getCursorColor(clientId) },
        }));
      },
      onCursorLeave: (clientId) => {
        setCursors((prev) => {
          const newCursors = { ...prev };
          delete newCursors[clientId];
          return newCursors;
        });
      },
    });

    return unsubscribe;
  }, []);

  const handleCardCursor = useCallback((cardId: number, x: number, y: number) => {
    wsManager.sendCursor(cardId, x, y);
  }, []);

  const handleCardCursorLeave = useCallback((cardId: number) => {
    wsManager.sendCursor(cardId, -1, -1);
  }, []);

  const getRemoteCursors = useCallback((cardId: number): RemoteCursor[] => {
    return Object.entries(cursors)
      .filter(([, pos]) => pos.cardId === cardId && pos.x >= 0)
      .map(([clientId, pos]) => ({ clientId, x: pos.x, y: pos.y, color: pos.color }));
  }, [cursors]);

  return {
    emojiUpdates,
    archiveUpdates,
    cursors,
    handleCardCursor,
    handleCardCursorLeave,
    getRemoteCursors,
  };
}
