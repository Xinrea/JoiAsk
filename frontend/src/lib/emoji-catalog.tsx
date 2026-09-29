'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { getInputEmojis, type InputEmoji } from '@/lib/api';
import { CONTENT_EMOJI_MAP } from '@joiask/content-markup';

const EmojiCatalogContext = createContext({
  emojis: [] as InputEmoji[],
  emojiMap: {} as Record<string, string>,
  loading: true,
  error: '',
  refresh: async () => {},
});

export function EmojiCatalogProvider({ children }: { children: React.ReactNode }) {
  const [emojis, setEmojis] = useState<InputEmoji[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const refresh = useCallback(async () => {
    try {
      const res = await getInputEmojis();
      if (res.code !== 200 || !Array.isArray(res.data)) throw new Error(res.message || '表情包加载失败');
      setEmojis(res.data);
      setError('');
    } catch {
      setError('表情包加载失败，请重试');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const onFocus = () => { void refresh(); };
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [refresh]);

  const emojiMap = useMemo(
    () => ({ ...CONTENT_EMOJI_MAP, ...Object.fromEntries(emojis.map(({ tag, url }) => [tag, url])) }),
    [emojis],
  );
  return (
    <EmojiCatalogContext.Provider value={{ emojis, emojiMap, loading, error, refresh }}>
      {children}
    </EmojiCatalogContext.Provider>
  );
}

export const useEmojiCatalog = () => useContext(EmojiCatalogContext);
