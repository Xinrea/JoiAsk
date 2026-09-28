'use client';

import { useState, useRef, useEffect } from 'react';
import { addEmoji, EmojiData } from '@/lib/api';

const EMOJIS = [
  { id: 0, value: '👍', asset: '/emojis/76.png' },
  { id: 1, value: '👎', asset: '/emojis/77.png' },
  { id: 2, value: '🤣', asset: null },
  { id: 3, value: '😭', asset: null },
  { id: 4, value: '😓', asset: null },
  { id: 5, value: '😬', asset: null },
  { id: 6, value: '🥳', asset: null },
  { id: 7, value: '😨', asset: null },
  { id: 8, value: '😠', asset: null },
  { id: 9, value: '💩', asset: '/emojis/59.png' },
  { id: 10, value: '💖', asset: '/emojis/66.png' },
  { id: 11, value: '🐵', asset: null },
  { id: 12, value: '❓', asset: null },
  { id: 13, value: '🫂', asset: '/emojis/49.png' },
  { id: 14, value: '🔘', asset: '/emojis/424.png' },
  { id: 15, value: '👅', asset: '/emojis/339.png' },
  { id: 16, value: '🥺', asset: '/emojis/111.png' },
  { id: 17, value: '👻', asset: '/emojis/187.png' },
  { id: 18, value: '😅', asset: null },
  { id: 19, value: '🌹', asset: '/emojis/63.png' },
];

const EMOJI_MAP: Record<string, string | null> = {
  '👍': '/emojis/76.png',
  '👎': '/emojis/77.png',
  '🤣': null,
  '😭': null,
  '😓': null,
  '😬': null,
  '🥳': null,
  '😨': null,
  '😠': null,
  '💩': '/emojis/59.png',
  '💖': '/emojis/66.png',
  '🐵': null,
  '❓': null,
  '🫂': '/emojis/49.png',
  '🔘': '/emojis/424.png',
  '👅': '/emojis/339.png',
  '🥺': '/emojis/111.png',
  '👻': '/emojis/187.png',
  '😅': null,
  '🌹': '/emojis/63.png',
};

interface EmojiPickerProps {
  questionId: number;
  emojis: EmojiData[];
}

export function EmojiPicker({ questionId, emojis: emojisProp }: EmojiPickerProps) {
  const [panelOpen, setPanelOpen] = useState(false);
  const [posted, setPosted] = useState(false);
  const pickerRef = useRef<HTMLDivElement>(null);

  // Sort emojis by count
  const emojis = [...emojisProp].sort((a, b) => b.count - a.count);

  // Close panel when clicking outside
  useEffect(() => {
    if (!panelOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        setPanelOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [panelOpen]);

  const handlePostEmoji = async (value: string) => {
    setPanelOpen(false);
    if (posted) return;

    try {
      const res = await addEmoji(questionId, value);
      if (res.code === 200) {
        setPosted(true);
        // WebSocket will handle the sync to all users
      }
    } catch (error) {
      console.error('Failed to post emoji:', error);
    }
  };

  return (
    <div ref={pickerRef} className="picker inline-block select-none relative">
      {panelOpen && (
        <div className="panel absolute bottom-12 right-4 w-[180px] z-[999] bg-card p-2.5 rounded border-2 border-dashed border-[var(--color-fabric-stitch)] grid grid-cols-4 shadow-lg">
          {EMOJIS.map(({ id, value, asset }) => (
            <div
              key={id}
              className="emoji-button flex text-base items-center justify-center cursor-pointer m-0.5 p-1 rounded hover:bg-accent transition-all duration-200"
              onClick={() => handlePostEmoji(value)}
            >
              {asset ? (
                <img src={asset} alt={value} height={16} width={16} />
              ) : (
                value
              )}
            </div>
          ))}
        </div>
      )}
      <div className="flex flex-row">
        {emojis.map(({ value, count }) => (
          <div
            key={value}
            className="emoji-button flex text-base items-center cursor-pointer m-0.5 px-1 border border-dashed border-[var(--color-fabric-stitch)] rounded hover:bg-accent transition-all duration-200"
            onClick={() => handlePostEmoji(value)}
          >
            {EMOJI_MAP[value] ? (
              <img src={EMOJI_MAP[value]!} alt={value} height={16} width={16} />
            ) : (
              value
            )}
            <span className="text-xs ml-0.5">{count}</span>
          </div>
        ))}
        <div
          className="emoji-button flex text-base items-center cursor-pointer m-0.5 px-1 border border-dashed border-[var(--color-fabric-stitch)] rounded hover:bg-accent transition-all duration-200"
          onClick={() => setPanelOpen(!panelOpen)}
        >
          <div
            className="h-6 w-6 bg-current"
            style={{
              maskImage: 'url(/emoji_add.svg)',
              WebkitMaskImage: 'url(/emoji_add.svg)',
            }}
          />
        </div>
      </div>
    </div>
  );
}
