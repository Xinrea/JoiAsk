'use client';

import { useState, useRef, useEffect, useCallback, MouseEvent, TouchEvent } from 'react';
import { X, ZoomIn, ZoomOut, RotateCw, Download, ChevronLeft, ChevronRight } from 'lucide-react';

interface ImagePreviewProps {
  src: string;
  images?: string[];
  currentIndex?: number;
  onClose: () => void;
  onIndexChange?: (index: number) => void;
}

export function ImagePreview({ src, images, currentIndex = 0, onClose, onIndexChange }: ImagePreviewProps) {
  const hasMultiple = (images?.length ?? 0) > 1;
  const [scale, setScale] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [rotation, setRotation] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const selectImage = useCallback((index: number) => {
    setScale(1);
    setPosition({ x: 0, y: 0 });
    setRotation(0);
    setIsDragging(false);
    onIndexChange?.(index);
  }, [onIndexChange]);

  const goToPrev = useCallback(() => {
    if (!images || !onIndexChange) return;
    const newIndex = currentIndex > 0 ? currentIndex - 1 : images.length - 1;
    selectImage(newIndex);
  }, [images, onIndexChange, currentIndex, selectImage]);

  const goToNext = useCallback(() => {
    if (!images || !onIndexChange) return;
    const newIndex = currentIndex < images.length - 1 ? currentIndex + 1 : 0;
    selectImage(newIndex);
  }, [images, onIndexChange, currentIndex, selectImage]);

  const dialogRef = useRef<HTMLDialogElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const thumbnailsRef = useRef<HTMLDivElement>(null);

  // Touch gesture refs
  const touchStartRef = useRef<{ x: number; y: number; time: number } | null>(null);
  const pinchStartDistRef = useRef<number | null>(null);
  const pinchStartScaleRef = useRef(1);

  // Lock body scroll when preview is open
  useEffect(() => {
    const dialog = dialogRef.current;
    dialog?.showModal();
    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      dialog?.close();
      document.body.style.overflow = originalOverflow;
    };
  }, []);

  useEffect(() => {
    thumbnailsRef.current?.children[currentIndex]?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [currentIndex]);

  // Handle zoom
  const handleZoom = useCallback((delta: number) => {
    setScale((prev) => {
      const newScale = prev + delta;
      return Math.max(0.1, Math.min(5, newScale));
    });
  }, []);

  // Handle wheel zoom with native event listener to prevent passive behavior
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleWheel = (e: globalThis.WheelEvent) => {
      e.preventDefault();
      const delta = e.deltaY > 0 ? -0.1 : 0.1;
      handleZoom(delta);
    };

    container.addEventListener('wheel', handleWheel, { passive: false });
    return () => container.removeEventListener('wheel', handleWheel);
  }, [handleZoom]);

  // Handle mouse drag start
  const handleMouseDown = (e: MouseEvent<HTMLDivElement>) => {
    if (e.button !== 0) return;
    setIsDragging(true);
    setDragStart({
      x: e.clientX - position.x,
      y: e.clientY - position.y,
    });
  };

  // Handle mouse drag
  const handleMouseMove = (e: MouseEvent<HTMLDivElement>) => {
    if (!isDragging) return;
    setPosition({
      x: e.clientX - dragStart.x,
      y: e.clientY - dragStart.y,
    });
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  // Touch: compute distance between two touches
  const getTouchDistance = (t1: React.Touch, t2: React.Touch) => {
    const dx = t1.clientX - t2.clientX;
    const dy = t1.clientY - t2.clientY;
    return Math.sqrt(dx * dx + dy * dy);
  };

  // Handle touch events for mobile (swipe + pinch + drag)
  const handleTouchStart = (e: TouchEvent<HTMLDivElement>) => {
    if (e.touches.length === 2) {
      // Pinch start
      pinchStartDistRef.current = getTouchDistance(e.touches[0], e.touches[1]);
      pinchStartScaleRef.current = scale;
      touchStartRef.current = null;
      setIsDragging(false);
    } else if (e.touches.length === 1) {
      // Record for swipe detection + drag
      touchStartRef.current = {
        x: e.touches[0].clientX,
        y: e.touches[0].clientY,
        time: Date.now(),
      };
      // Only enable drag when zoomed in
      if (scale > 1) {
        setIsDragging(true);
        setDragStart({
          x: e.touches[0].clientX - position.x,
          y: e.touches[0].clientY - position.y,
        });
      }
    }
  };

  const handleTouchMove = (e: TouchEvent<HTMLDivElement>) => {
    if (e.touches.length === 2 && pinchStartDistRef.current !== null) {
      // Pinch zoom
      const currentDist = getTouchDistance(e.touches[0], e.touches[1]);
      const ratio = currentDist / pinchStartDistRef.current;
      const newScale = Math.max(0.1, Math.min(5, pinchStartScaleRef.current * ratio));
      setScale(newScale);
      return;
    }
    if (!isDragging || e.touches.length !== 1) return;
    setPosition({
      x: e.touches[0].clientX - dragStart.x,
      y: e.touches[0].clientY - dragStart.y,
    });
  };

  const handleTouchEnd = (e: TouchEvent<HTMLDivElement>) => {
    // Pinch end
    if (pinchStartDistRef.current !== null && e.touches.length < 2) {
      pinchStartDistRef.current = null;
      setIsDragging(false);
      touchStartRef.current = null;
      return;
    }

    setIsDragging(false);

    // Swipe detection (only when not zoomed in)
    if (scale <= 1 && touchStartRef.current && hasMultiple) {
      const dx = (e.changedTouches[0]?.clientX ?? 0) - touchStartRef.current.x;
      const dy = (e.changedTouches[0]?.clientY ?? 0) - touchStartRef.current.y;
      const dt = Date.now() - touchStartRef.current.time;
      const absDx = Math.abs(dx);
      const absDy = Math.abs(dy);

      // Horizontal swipe: >60px, mostly horizontal, within 300ms
      if (absDx > 60 && absDx > absDy * 1.5 && dt < 300) {
        if (dx > 0) {
          goToPrev();
        } else {
          goToNext();
        }
      }
    }
    touchStartRef.current = null;
  };

  // Handle rotation
  const handleRotate = () => {
    setRotation((prev) => (prev + 90) % 360);
  };

  // Handle download
  const handleDownload = async () => {
    try {
      const response = await fetch(src);
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `image-${Date.now()}.jpg`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Download failed:', error);
    }
  };

  // Handle reset
  const handleReset = () => {
    setScale(1);
    setPosition({ x: 0, y: 0 });
    setRotation(0);
  };

  // Handle keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      switch (e.key) {
        case '+':
        case '=':
          handleZoom(0.2);
          break;
        case '-':
          handleZoom(-0.2);
          break;
        case 'r':
        case 'R':
          handleRotate();
          break;
        case '0':
          handleReset();
          break;
        case 'ArrowLeft':
          if (hasMultiple) goToPrev();
          break;
        case 'ArrowRight':
          if (hasMultiple) goToNext();
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose, hasMultiple, goToPrev, goToNext, handleZoom]);

  const toolbarButtonClass = "relative flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-white/10 transition-colors before:absolute before:-inset-1 before:content-[''] hover:bg-white/20 active:bg-white/30 focus-visible:outline-2 focus-visible:outline-white";

  return (
    <dialog
      ref={dialogRef}
      aria-label="投稿图片"
      onCancel={(event) => { event.preventDefault(); onClose(); }}
      className="fixed inset-0 m-0 h-dvh max-h-none w-full max-w-none overflow-hidden border-0 bg-[#161412] p-0 text-white backdrop:bg-black/70"
    >
      <section aria-label="图片浏览" className="flex h-full min-h-0 flex-col">
        <div className="flex shrink-0 items-center justify-between gap-2 px-3 py-2 sm:px-5" style={{ paddingTop: 'max(0.5rem, env(safe-area-inset-top))' }}>
          <button onClick={handleReset} className="rounded-lg px-2 py-2 text-sm text-white/75 hover:bg-white/10" title="重置 (0)">
            {Math.round(scale * 100)}%
          </button>
          <div className="flex gap-2">
            <button onClick={() => handleZoom(0.2)} className={toolbarButtonClass} title="放大 (+)" aria-label="放大">
              <ZoomIn className="h-4.5 w-4.5" />
            </button>
            <button onClick={() => handleZoom(-0.2)} className={toolbarButtonClass} title="缩小 (-)" aria-label="缩小">
              <ZoomOut className="h-4.5 w-4.5" />
            </button>
            <button onClick={handleRotate} className={toolbarButtonClass} title="旋转 (R)" aria-label="旋转">
              <RotateCw className="h-4.5 w-4.5" />
            </button>
            <button onClick={handleDownload} className={toolbarButtonClass} title="下载" aria-label="下载图片">
              <Download className="h-4.5 w-4.5" />
            </button>
            <button onClick={onClose} className={toolbarButtonClass} title="关闭 (ESC)" aria-label="关闭图片浏览" autoFocus>
              <X className="h-4.5 w-4.5" />
            </button>
          </div>
        </div>

        <div className="relative min-h-0 flex-1">
          <div
            ref={containerRef}
            className="absolute inset-0 flex touch-none items-center justify-center overflow-hidden px-3 py-2 sm:px-16 sm:py-4"
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            onMouseLeave={handleMouseUp}
            onTouchStart={handleTouchStart}
            onTouchMove={handleTouchMove}
            onTouchEnd={handleTouchEnd}
            onTouchCancel={() => { setIsDragging(false); touchStartRef.current = null; pinchStartDistRef.current = null; }}
            style={{ cursor: isDragging ? 'grabbing' : 'grab' }}
          >
            <img
              src={src}
              alt={`投稿图片 ${currentIndex + 1}`}
              className="h-full w-full select-none object-contain"
              draggable={false}
              style={{
                transform: `translate(${position.x}px, ${position.y}px) scale(${scale}) rotate(${rotation}deg)`,
                transition: isDragging ? 'none' : 'transform 0.1s ease-out',
              }}
            />
          </div>
          {hasMultiple && (
            <>
              <button onClick={goToPrev} className="absolute left-2 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 hover:bg-black/75 sm:left-4" title="上一张 (←)" aria-label="上一张">
                <ChevronLeft className="h-6 w-6" />
              </button>
              <button onClick={goToNext} className="absolute right-2 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-black/50 hover:bg-black/75 sm:right-4" title="下一张 (→)" aria-label="下一张">
                <ChevronRight className="h-6 w-6" />
              </button>
            </>
          )}
        </div>

        <div className="shrink-0 px-3 pb-3 pt-2 sm:px-5" style={{ paddingBottom: 'max(0.75rem, env(safe-area-inset-bottom))' }}>
          {hasMultiple && (
            <div ref={thumbnailsRef} aria-label="选择图片" className="flex gap-2 overflow-x-auto overscroll-contain py-1">
              {images!.map((image, index) => (
                <button
                  key={`${image}-${index}`}
                  onClick={() => selectImage(index)}
                  aria-label={`查看第 ${index + 1} 张图片`}
                  aria-pressed={index === currentIndex}
                  className={`h-12 w-12 shrink-0 overflow-hidden rounded-lg border-2 transition-opacity sm:h-16 sm:w-16 ${index === currentIndex ? 'border-white opacity-100' : 'border-transparent opacity-50 hover:opacity-100'}`}
                >
                  <img src={image} alt="" className="h-full w-full object-cover" draggable={false} />
                </button>
              ))}
            </div>
          )}
          <div className="mt-2 flex items-center justify-between gap-3 text-xs text-white/60">
            <span className="shrink-0">{currentIndex + 1} / {images?.length ?? 1}</span>
            <span className="hidden sm:inline">拖拽移动 · 滚轮缩放 · R 旋转 · 0 重置{hasMultiple ? ' · ←/→ 切换' : ''}</span>
            <span className="sm:hidden">双指缩放{hasMultiple ? ' · 左右滑动切换' : ''}</span>
          </div>
        </div>
      </section>
    </dialog>
  );
}
