'use client';

import { useState, useRef, useEffect } from 'react';
import { useEmojiCatalog } from '@/lib/emoji-catalog';

interface InputEmojiPickerProps {
  onSelect: (tag: string) => void;
}

export function InputEmojiPicker({ onSelect }: InputEmojiPickerProps) {
  const [panelOpen, setPanelOpen] = useState(false);
  const { emojis: inputEmojis, loading, error, refresh } = useEmojiCatalog();
  const pickerRef = useRef<HTMLDivElement>(null);

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

  const handleSelect = (tag: string) => {
    onSelect(tag);
    setPanelOpen(false);
  };

  return (
    <div ref={pickerRef} className="relative inline-block">
      <button
        type="button"
        onClick={() => {
          if (!panelOpen) void refresh();
          setPanelOpen(!panelOpen);
        }}
        className="p-2 text-primary hover:bg-accent rounded transition-all duration-200 border-2 border-dashed border-[var(--color-fabric-stitch)]"
        title="插入表情"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10"/>
          <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
          <line x1="9" y1="9" x2="9.01" y2="9"/>
          <line x1="15" y1="9" x2="15.01" y2="9"/>
        </svg>
      </button>
      {panelOpen && (
        <div className="absolute bottom-full mb-2 right-0 w-[280px] max-h-72 overflow-y-auto z-[999] bg-card p-2 rounded border-2 border-dashed border-[var(--color-fabric-stitch)] grid grid-cols-5 gap-1 shadow-lg">
          {loading && <p className="col-span-5 text-sm">加载中...</p>}
          {error && <button type="button" className="col-span-5 text-sm" onClick={() => void refresh()}>{error}</button>}
          {!loading && !error && inputEmojis.length === 0 && <p className="col-span-5 text-sm">暂无表情包</p>}
          {inputEmojis.map(({ tag, url }) => (
            <button
              type="button"
              key={tag}
              className="flex flex-col items-center justify-center cursor-pointer p-1 rounded hover:bg-accent transition-all duration-200"
              onClick={() => handleSelect(tag)}
              title={tag}
            >
              <img src={url} alt={tag} className="w-10 h-10 object-contain" />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
