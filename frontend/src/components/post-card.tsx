'use client';

import { useState, useRef, useEffect, MouseEvent, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Question, updateQuestion, EmojiData } from '@/lib/api';
import { EmojiPicker } from './emoji-picker';

interface RemoteCursor {
  clientId: string;
  x: number;
  y: number;
  color: string;
}

interface PostCardProps {
  data: Question;
  isLoggedIn?: boolean;
  onArchiveUpdate?: (id: number) => void;
  emojiUpdates?: Record<number, EmojiData[]>;
  archiveUpdates?: Record<number, boolean>;
  remoteCursors?: RemoteCursor[];
  onCursorMove?: (x: number, y: number) => void;
  onCursorLeave?: () => void;
}

const EMOJI_MAP: Record<string, string> = {
  '[轴伊Joi收藏集动态表情包_跑了]': '/joi-emojis/paole.webp',
  '[轴伊Joi收藏集动态表情包_鞠躬]': '/joi-emojis/jugong.webp',
  '[轴伊Joi收藏集动态表情包_摇你]': '/joi-emojis/yaoni.webp',
  '[轴伊Joi收藏集动态表情包_愤怒]': '/joi-emojis/fennu.webp',
  '[轴伊Joi收藏集动态表情包_猴]': '/joi-emojis/hou.webp',
  '[轴伊Joi收藏集动态表情包_NO]': '/joi-emojis/no.webp',
  '[轴伊Joi收藏集动态表情包_贴贴]': '/joi-emojis/tietie.webp',
  '[轴伊Joi收藏集动态表情包_呆]': '/joi-emojis/dai.webp',
  '[轴伊Joi收藏集动态表情包_唔唔]': '/joi-emojis/wuwu.webp',
  '[轴伊Joi收藏集动态表情包_啊这]': '/joi-emojis/azhe.webp',
  '[轴伊Joi收藏集动态表情包_失落]': '/joi-emojis/shiluo.webp',
  '[轴伊Joi收藏集动态表情包_神气]': '/joi-emojis/shenqi.webp',
  '[轴伊Joi收藏集动态表情包_怎么这样]': '/joi-emojis/zenmezhyang.webp',
  '[轴伊Joi收藏集动态表情包_睡觉]': '/joi-emojis/shuijiao.webp',
  '[轴伊Joi收藏集动态表情包_爆]': '/joi-emojis/bao.webp',
};

function renderContent(content: string): string {
  let rendered = content.replace(/｛/g, '{').replace(/｝/g, '}');
  rendered = rendered.replace(/</g, '&lt;').replace(/>/g, '&gt;');
  rendered = rendered.replace(
    /\{\{([\s\S]*?)\}\}/g,
    '<span style="color:#5a4e3f;background:#5a4e3f;transition:all 260ms ease;border-radius:2px;" onmouseover="this.style.color=\'#faf7f2\'" onmouseout="this.style.color=\'#5a4e3f\'">$1</span>'
  );
  // Replace URLs with links
  rendered = rendered.replace(
    /https?:\/\/[^\s<]+/g,
    (match) => `<a href="${match}" target="_blank" rel="noopener noreferrer" style="color:#00a1d6;text-decoration:underline;">${match}</a>`
  );
  // Replace BV codes with bilibili links (only if not already linked)
  rendered = rendered.replace(
    /(?<!href="[^"]*?)(?<!">)BV[a-zA-Z0-9]{10}(?![^<]*<\/a>)/g,
    (match) => `<a href="https://www.bilibili.com/video/${match}" target="_blank" rel="noopener noreferrer" style="color:#00a1d6;text-decoration:underline;">${match}</a>`
  );
  // Replace emoji tags with images
  for (const [tag, url] of Object.entries(EMOJI_MAP)) {
    const escapedTag = tag.replace(/[[\]]/g, '\\$&');
    rendered = rendered.replace(
      new RegExp(escapedTag, 'g'),
      `<img src="${url}" alt="${tag}" style="display:inline-block;height:64px;width:64px;vertical-align:middle;margin:0 2px;object-fit:contain;" />`
    );
  }
  return rendered;
}

function formatTime(dateString: string): string {
  const postTimestamp = new Date(dateString).getTime() / 1000;
  let time = Math.floor(Date.now() / 1000 - postTimestamp);
  let unit = '秒';

  if (time > 60) {
    time = Math.floor(time / 60);
    unit = '分钟';
    if (time > 60) {
      time = Math.floor(time / 60);
      unit = '小时';
      if (time > 24) {
        return new Date(postTimestamp * 1000).toLocaleDateString();
      }
    }
  }
  return `${time}${unit} 前`;
}

export function PostCard({ data, isLoggedIn = false, emojiUpdates, archiveUpdates, remoteCursors = [], onCursorMove, onCursorLeave }: PostCardProps) {
  const router = useRouter();
  const cardRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [scrollEnd, setScrollEnd] = useState(true);
  const [showImageModal, setShowImageModal] = useState(false);
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [isArchived, setIsArchived] = useState(data.is_archive);

  // 3D tilt state (using ref to avoid re-renders)
  const transformRef = useRef({
    rx: 0,
    ry: 0,
    px: 50,
    py: 50,
    hyp: 0.03,
  });

  const maxDegree = 30;
  const recoverIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Parse emojis
  const [emojis, setEmojis] = useState<EmojiData[]>(() => {
    try {
      return data.emojis ? JSON.parse(data.emojis) : [];
    } catch {
      return [];
    }
  });

  // Update emojis from SSE
  useEffect(() => {
    if (emojiUpdates && emojiUpdates[data.id]) {
      setEmojis(emojiUpdates[data.id]);
    }
  }, [emojiUpdates, data.id]);

  // Update archive status from SSE
  useEffect(() => {
    if (archiveUpdates && archiveUpdates[data.id]) {
      setIsArchived(true);
    }
  }, [archiveUpdates, data.id]);

  useEffect(() => {
    if (contentRef.current) {
      const el = contentRef.current;
      setScrollEnd(el.scrollTop + el.clientHeight >= el.scrollHeight);
    }
  }, []);

  const imageList = data.images ? data.images.split(';').filter(Boolean) : [];

  // Throttle cursor updates to reduce network traffic
  const lastCursorUpdate = useRef(0);
  const throttledCursorMove = useCallback((x: number, y: number) => {
    const now = Date.now();
    if (now - lastCursorUpdate.current > 50) { // 50ms throttle
      lastCursorUpdate.current = now;
      onCursorMove?.(x, y);
    }
  }, [onCursorMove]);

  const handleMouseMove = (e: MouseEvent<HTMLDivElement>) => {
    if (!cardRef.current) return;

    // Skip all updates when mouse button is pressed (during text selection drag)
    if (e.buttons !== 0) return;

    const rect = cardRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    const w = rect.width;
    const h = rect.height;

    const px = (x / w) * 100;
    const py = (y / h) * 100;
    const rx = -((x / w) * maxDegree - maxDegree / 2);
    const ry = (y / h) * maxDegree - maxDegree / 2;
    const hyp =
      Math.sqrt(Math.pow(x - w / 2, 2) + Math.pow(y - h / 2, 2)) /
      Math.sqrt(Math.pow(w / 2, 2) + Math.pow(h / 2, 2));

    // Store current values in ref
    transformRef.current = { rx, ry, px, py, hyp };

    // Update DOM directly to avoid re-renders that clear text selection
    cardRef.current.style.transform = `perspective(1500px) rotateX(${ry}deg) rotateY(${rx}deg)`;
    cardRef.current.style.setProperty('--px', `${px}%`);
    cardRef.current.style.setProperty('--py', `${py}%`);
    cardRef.current.style.setProperty('--hyp', hyp.toString());

    // Send cursor position
    throttledCursorMove(px, py);
  };

  const handleMouseLeave = () => {
    if (recoverIntervalRef.current) {
      clearInterval(recoverIntervalRef.current);
    }

    recoverIntervalRef.current = setInterval(() => {
      if (!cardRef.current) return;

      const t = transformRef.current;
      let complete = true;

      // Gradually move rx towards 0
      if (Math.abs(t.rx) > 0.5) {
        t.rx = t.rx > 0 ? t.rx - 1 : t.rx + 1;
        complete = false;
      } else {
        t.rx = 0;
      }

      // Gradually move ry towards 0
      if (Math.abs(t.ry) > 0.5) {
        t.ry = t.ry > 0 ? t.ry - 1 : t.ry + 1;
        complete = false;
      } else {
        t.ry = 0;
      }

      // Gradually move hyp towards 0.03
      if (Math.abs(t.hyp - 0.03) > 0.005) {
        t.hyp = t.hyp > 0.03 ? t.hyp - 0.01 : t.hyp + 0.01;
        complete = false;
      } else {
        t.hyp = 0.03;
      }

      // Update DOM
      cardRef.current.style.transform = `perspective(1500px) rotateX(${t.ry}deg) rotateY(${t.rx}deg)`;
      cardRef.current.style.setProperty('--hyp', t.hyp.toString());

      if (complete && recoverIntervalRef.current) {
        clearInterval(recoverIntervalRef.current);
      }
    }, 16);

    // Notify cursor leave
    onCursorLeave?.();
  };

  const handleMouseEnter = () => {
    if (recoverIntervalRef.current) {
      clearInterval(recoverIntervalRef.current);
    }
  };

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const el = e.currentTarget;
    setScrollEnd(el.scrollTop + el.clientHeight >= el.scrollHeight);
  };

  const handleArchive = async () => {
    try {
      const res = await updateQuestion(data.id, {
        tag_id: data.tag_id,
        is_hide: data.is_hide,
        is_rainbow: data.is_rainbow,
        is_archive: true,
        is_publish: data.is_publish,
      });
      if (res.code === 200) {
        setIsArchived(true);
      } else {
        alert(res.message);
      }
    } catch (error) {
      console.error('Failed to archive:', error);
    }
  };

  const handleTagClick = () => {
    router.push(`/tags/${data.tag_id}?name=${encodeURIComponent(data.tag.tag_name)}`);
  };

  return (
    <div className="flex flex-col w-full p-6 justify-center items-center">
      <div className="w-full md:w-auto flex flex-col items-center">
        <div
          ref={cardRef}
          className="card-wrap relative h-[346px] w-full md:w-[600px] rounded-md overflow-hidden"
          style={{
            transformOrigin: 'center',
            transform: 'perspective(1500px) rotateX(0deg) rotateY(0deg)',
            transformStyle: 'preserve-3d',
            ['--px' as string]: '50%',
            ['--py' as string]: '50%',
            ['--hyp' as string]: 0.03,
          }}
          onMouseMove={handleMouseMove}
          onMouseLeave={handleMouseLeave}
          onMouseEnter={handleMouseEnter}
        >
          <div
            className={`card h-full w-full fabric-card shadow-lg p-4 text-muted-foreground flex flex-col justify-between border-2 border-dashed border-[var(--fabric-stitch)] ${
              data.is_rainbow ? 'special' : ''
            }`}
          >
            {/* Watermark */}
            <div
              className="watermark cursor-pointer z-[5]"
              onClick={handleTagClick}
            >
              #{data.tag?.tag_name || '未分类'}
            </div>

            {/* Header */}
            <div className="text-left text-sm h-6">
              #{data.id} | {formatTime(data.created_at)}
            </div>

            {/* Content */}
            <div
              ref={contentRef}
              className={`card-content flex flex-col my-4 justify-start overflow-y-auto relative text-left text-foreground h-full resize-none bg-transparent text-base z-[3] ${
                !scrollEnd ? 'border-b-2 border-dashed border-primary/30' : ''
              }`}
              style={{
                whiteSpace: 'pre-wrap',
                fontFamily: 'system-ui, Fangsong, STFangsong, serif',
                msOverflowStyle: 'none',
                scrollbarWidth: 'none',
              }}
              onScroll={handleScroll}
            >
              <div>
                {data.images_num > 0 && (
                  <div
                    className="stamp float-right w-20 h-20 cursor-pointer p-2.5 text-center transition-transform duration-260 hover:translate-x-[-10px] hover:translate-y-[10px] relative z-[1]"
                    onClick={() => setShowImageModal(true)}
                    style={{
                      background: 'radial-gradient(transparent 0px, transparent 4px, var(--card) 4px, var(--card))',
                      backgroundSize: '20px 20px',
                      backgroundPosition: '-10px -10px',
                      filter: 'drop-shadow(-5px -5px 10px rgba(139, 111, 71, 0.1))',
                    }}
                  >
                    <div
                      className="stamp__content bg-card w-full h-full relative"
                      style={{
                        backgroundImage: 'url(/image_stamp.png)',
                        backgroundSize: 'cover',
                      }}
                    >
                      <span className="absolute bottom-2 right-2 bg-black/60 text-white w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold">
                        {data.images_num}
                      </span>
                    </div>
                  </div>
                )}
                <span dangerouslySetInnerHTML={{ __html: renderContent(data.content) }} />
                {isArchived && (
                  <div
                    className="watermark archived pointer-events-none"
                    style={{
                      color: 'rgba(255, 0, 0, 0.51)',
                      boxShadow: '0 0 0 3px rgba(255, 0, 0, 0.4), 0 0 0 2px rgba(255, 0, 0, 0.4) inset',
                      transform: 'rotate(10deg) scale(1.2)',
                      top: '70%',
                      left: '20%',
                    }}
                  >
                    已归档
                  </div>
                )}
              </div>
            </div>

            {/* Footer */}
            <div className="text-right text-sm h-6 flex justify-end items-center">
              <EmojiPicker
                questionId={data.id}
                emojis={emojis}
              />
              {isLoggedIn && !isArchived && (
                <div
                  className="cursor-pointer inline-block select-none ml-2 border-2 border-dashed border-[var(--fabric-stitch)] rounded px-1 text-primary hover:bg-accent transition-all duration-200"
                  onClick={handleArchive}
                >
                  归档此卡
                </div>
              )}
            </div>
          </div>

          {/* Rainbow textures */}
          {data.is_rainbow && (
            <>
              <div
                className="texture-illusion absolute w-full h-full top-0 left-0 pointer-events-none"
                style={{
                  backgroundImage: `url(/vmaxbg.jpg), repeating-linear-gradient(0deg, rgb(255, 119, 115) 5%, rgba(255, 237, 95, 1) 10%, rgba(168, 255, 95, 1) 15%, rgba(131, 255, 247, 1) 20%, rgba(120, 148, 255, 1) 25%, rgb(216, 117, 255) 30%, rgb(255, 119, 115) 35%), repeating-linear-gradient(133deg, #0e152e 0%, hsl(180, 10%, 60%) 7.6%, hsl(180, 29%, 66%) 9%, hsl(180, 10%, 60%) 10.4%, #0e152e 20%, #0e152e 24%), radial-gradient(farthest-corner circle at var(--px) var(--py), rgba(0, 0, 0, 0.1) 12%, rgba(0, 0, 0, 0.15) 20%, rgba(0, 0, 0, 0.25) 120%)`,
                  backgroundPosition: `center, 0% var(--py), var(--px) var(--py), var(--px) var(--py)`,
                  filter: `brightness(calc(var(--hyp) * 0.3 + 0.3)) contrast(2) saturate(1.5)`,
                  backgroundBlendMode: 'exclusion, hue, hard-light',
                  backgroundSize: '50%',
                  mixBlendMode: 'color-dodge',
                  opacity: 0.5,
                }}
              />
              <div
                className="texture-noise absolute w-full h-full top-0 left-0 pointer-events-none"
                style={{
                  backgroundImage: 'url(/texture.svg)',
                  mixBlendMode: 'color-dodge',
                  opacity: 0.3,
                }}
              />
            </>
          )}

          {/* Remote cursors */}
          {remoteCursors.map((cursor) => (
            <div
              key={cursor.clientId}
              className="absolute pointer-events-none z-50 transition-all duration-75"
              style={{
                left: `${cursor.x}%`,
                top: `${cursor.y}%`,
                transform: 'translate(-50%, -50%)',
              }}
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill={cursor.color}
                opacity={0.6}
                style={{ filter: 'drop-shadow(0 1px 1px rgba(0,0,0,0.2))' }}
              >
                <path d="M5.5 3.21V20.8c0 .45.54.67.85.35l4.86-4.86a.5.5 0 0 1 .35-.15h6.87c.48 0 .72-.58.38-.92L6.35 2.85a.5.5 0 0 0-.85.36Z" />
              </svg>
            </div>
          ))}
        </div>
      </div>

      {/* Image Modal */}
      {showImageModal && (
        <div
          className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50"
          onClick={() => setShowImageModal(false)}
        >
          <div className="p-2 w-[300px]" onClick={(e) => e.stopPropagation()}>
            <div className="grid grid-cols-3 gap-3">
              {imageList.map((image, index) => (
                <div
                  key={index}
                  className="relative w-full pb-[100%] overflow-hidden rounded-lg cursor-pointer"
                >
                  <img
                    src={image}
                    alt={`Image ${index + 1}`}
                    className="absolute top-0 left-0 w-full h-full object-cover transition-transform hover:scale-105"
                    onClick={() => setPreviewImage(image)}
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Full Image Preview */}
      {previewImage && (
        <div
          className="fixed inset-0 bg-black z-[9999] flex items-center justify-center cursor-pointer"
          onClick={() => setPreviewImage(null)}
        >
          <img
            src={previewImage}
            alt="Preview"
            className="max-w-full max-h-full object-contain"
          />
        </div>
      )}

      <style jsx>{`
        .card-content::-webkit-scrollbar {
          display: none;
        }
        .special {
          background-image: linear-gradient(
            133deg,
            rgba(220, 93, 231, 0.3) 0%,
            rgba(127, 239, 189, 0.3) 33%,
            rgba(255, 245, 137, 0.3) 66%,
            rgba(236, 11, 67, 0.3) 100%
          );
        }
        .watermark {
          position: absolute;
          top: 20px;
          right: 6px;
          font-family: system-ui;
          color: rgba(139, 111, 71, 0.3);
          box-shadow: 0 0 0 3px rgba(139, 111, 71, 0.3), 0 0 0 2px rgba(139, 111, 71, 0.3) inset;
          border: 2px dashed transparent;
          border-radius: 4px;
          display: inline-block;
          padding: 5px 2px;
          line-height: 22px;
          font-size: 24px;
          text-transform: uppercase;
          text-align: center;
          width: 155px;
          transform: rotate(-5deg);
        }
      `}</style>
    </div>
  );
}
