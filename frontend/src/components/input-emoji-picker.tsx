'use client';

import { useState, useRef, useEffect } from 'react';

const INPUT_EMOJIS = [
  { name: '跑了', tag: '[轴伊Joi收藏集动态表情包_跑了]', url: '/joi-emojis/paole.webp' },
  { name: '鞠躬', tag: '[轴伊Joi收藏集动态表情包_鞠躬]', url: '/joi-emojis/jugong.webp' },
  { name: '摇你', tag: '[轴伊Joi收藏集动态表情包_摇你]', url: '/joi-emojis/yaoni.webp' },
  { name: '愤怒', tag: '[轴伊Joi收藏集动态表情包_愤怒]', url: '/joi-emojis/fennu.webp' },
  { name: '猴', tag: '[轴伊Joi收藏集动态表情包_猴]', url: '/joi-emojis/hou.webp' },
  { name: 'NO', tag: '[轴伊Joi收藏集动态表情包_NO]', url: '/joi-emojis/no.webp' },
  { name: '贴贴', tag: '[轴伊Joi收藏集动态表情包_贴贴]', url: '/joi-emojis/tietie.webp' },
  { name: '呆', tag: '[轴伊Joi收藏集动态表情包_呆]', url: '/joi-emojis/dai.webp' },
  { name: '唔唔', tag: '[轴伊Joi收藏集动态表情包_唔唔]', url: '/joi-emojis/wuwu.webp' },
  { name: '啊这', tag: '[轴伊Joi收藏集动态表情包_啊这]', url: '/joi-emojis/azhe.webp' },
  { name: '失落', tag: '[轴伊Joi收藏集动态表情包_失落]', url: '/joi-emojis/shiluo.webp' },
  { name: '神气', tag: '[轴伊Joi收藏集动态表情包_神气]', url: '/joi-emojis/shenqi.webp' },
  { name: '怎么这样', tag: '[轴伊Joi收藏集动态表情包_怎么这样]', url: '/joi-emojis/zenmezhyang.webp' },
  { name: '睡觉', tag: '[轴伊Joi收藏集动态表情包_睡觉]', url: '/joi-emojis/shuijiao.webp' },
  { name: '爆', tag: '[轴伊Joi收藏集动态表情包_爆]', url: '/joi-emojis/bao.webp' },
];

interface InputEmojiPickerProps {
  onSelect: (tag: string) => void;
}

export function InputEmojiPicker({ onSelect }: InputEmojiPickerProps) {
  const [panelOpen, setPanelOpen] = useState(false);
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
        onClick={() => setPanelOpen(!panelOpen)}
        className="p-2 text-primary hover:bg-accent rounded transition-all duration-200 border-2 border-dashed border-[var(--fabric-stitch)]"
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
        <div className="absolute bottom-full mb-2 right-0 w-[280px] z-[999] bg-card p-2 rounded border-2 border-dashed border-[var(--fabric-stitch)] grid grid-cols-5 gap-1 shadow-lg">
          {INPUT_EMOJIS.map(({ name, tag, url }) => (
            <div
              key={tag}
              className="flex flex-col items-center justify-center cursor-pointer p-1 rounded hover:bg-accent transition-all duration-200"
              onClick={() => handleSelect(tag)}
              title={name}
            >
              <img src={url} alt={name} className="w-10 h-10 object-contain" />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
