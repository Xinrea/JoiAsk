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
  const hasMultiple = images && images.length > 1;

  const goToPrev = useCallback(() => {
    if (!images || !onIndexChange) return;
    const newIndex = currentIndex > 0 ? currentIndex - 1 : images.length - 1;
    onIndexChange(newIndex);
  }, [images, onIndexChange, currentIndex]);

  const goToNext = useCallback(() => {
    if (!images || !onIndexChange) return;
    const newIndex = currentIndex < images.length - 1 ? currentIndex + 1 : 0;
    onIndexChange(newIndex);
  }, [images, onIndexChange, currentIndex]);

  const [scale, setScale] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [rotation, setRotation] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const imageRef = useRef<HTMLImageElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Touch gesture refs
  const touchStartRef = useRef<{ x: number; y: number; time: number } | null>(null);
  const pinchStartDistRef = useRef<number | null>(null);
  const pinchStartScaleRef = useRef(1);

  // Reset on image change
  useEffect(() => {
    setScale(1);
    setPosition({ x: 0, y: 0 });
    setRotation(0);
  }, [src]);

  // Lock body scroll when preview is open
  useEffect(() => {
    const originalOverflow = document.body.style.overflow;
    const originalTouchAction = document.body.style.touchAction;
    document.body.style.overflow = 'hidden';
    document.body.style.touchAction = 'none';
    return () => {
      document.body.style.overflow = originalOverflow;
      document.body.style.touchAction = originalTouchAction;
    };
  }, []);

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
  const getTouchDistance = (t1: globalThis.Touch, t2: globalThis.Touch) => {
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
      switch (e.key) {
        case 'Escape':
          onClose();
          break;
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

  return (
    <div
      ref={containerRef}
      className="fixed inset-0 z-[100] bg-black/95 select-none"
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
    >
      {/* Toolbar */}
      <div className="absolute top-4 right-4 z-20 flex gap-2">
        <button
          onClick={() => handleZoom(0.2)}
          className="p-2.5 sm:p-2 bg-white/10 hover:bg-white/20 active:bg-white/30 rounded-lg backdrop-blur-sm transition-colors"
          title="放大 (+)"
        >
          <ZoomIn className="w-5 h-5 text-white" />
        </button>
        <button
          onClick={() => handleZoom(-0.2)}
          className="p-2.5 sm:p-2 bg-white/10 hover:bg-white/20 active:bg-white/30 rounded-lg backdrop-blur-sm transition-colors"
          title="缩小 (-)"
        >
          <ZoomOut className="w-5 h-5 text-white" />
        </button>
        <button
          onClick={handleRotate}
          className="p-2.5 sm:p-2 bg-white/10 hover:bg-white/20 active:bg-white/30 rounded-lg backdrop-blur-sm transition-colors"
          title="旋转 (R)"
        >
          <RotateCw className="w-5 h-5 text-white" />
        </button>
        <button
          onClick={handleDownload}
          className="p-2.5 sm:p-2 bg-white/10 hover:bg-white/20 active:bg-white/30 rounded-lg backdrop-blur-sm transition-colors"
          title="下载"
        >
          <Download className="w-5 h-5 text-white" />
        </button>
        <button
          onClick={onClose}
          className="p-2.5 sm:p-2 bg-white/10 hover:bg-white/20 active:bg-white/30 rounded-lg backdrop-blur-sm transition-colors"
          title="关闭 (ESC)"
        >
          <X className="w-5 h-5 text-white" />
        </button>
      </div>

      {/* Scale indicator */}
      <div className="absolute top-4 left-4 px-3 py-2 bg-white/10 rounded-lg backdrop-blur-sm text-white text-sm">
        {Math.round(scale * 100)}%
      </div>

      {/* Prev button */}
      {hasMultiple && (
        <button
          onClick={(e) => { e.stopPropagation(); goToPrev(); }}
          className="absolute left-2 sm:left-4 top-1/2 -translate-y-1/2 z-10 p-3 sm:p-2 bg-white/10 hover:bg-white/20 active:bg-white/30 rounded-full backdrop-blur-sm transition-colors"
          title="上一张 (←)"
        >
          <ChevronLeft className="w-6 h-6 text-white" />
        </button>
      )}

      {/* Next button */}
      {hasMultiple && (
        <button
          onClick={(e) => { e.stopPropagation(); goToNext(); }}
          className="absolute right-2 sm:right-4 top-1/2 -translate-y-1/2 z-10 p-3 sm:p-2 bg-white/10 hover:bg-white/20 active:bg-white/30 rounded-full backdrop-blur-sm transition-colors"
          title="下一张 (→)"
        >
          <ChevronRight className="w-6 h-6 text-white" />
        </button>
      )}

      {/* Image counter */}
      {hasMultiple && (
        <div className="absolute bottom-14 left-1/2 transform -translate-x-1/2 px-3 py-1.5 bg-white/10 rounded-lg backdrop-blur-sm text-white text-sm">
          {currentIndex + 1} / {images.length}
        </div>
      )}

      {/* Image container */}
      <div
        className="relative w-full h-full flex items-center justify-center overflow-hidden"
        onMouseDown={handleMouseDown}
        onTouchStart={handleTouchStart}
        style={{ cursor: isDragging ? 'grabbing' : 'grab' }}
      >
        <img
          ref={imageRef}
          src={src}
          alt="Preview"
          className="max-w-[90vw] max-h-[85vh] sm:max-w-none sm:max-h-none object-contain select-none"
          draggable={false}
          style={{
            transform: `translate(${position.x}px, ${position.y}px) scale(${scale}) rotate(${rotation}deg)`,
            transition: isDragging ? 'none' : 'transform 0.1s ease-out',
          }}
        />
      </div>

      {/* Instructions - desktop only */}
      <div className="hidden sm:block absolute bottom-4 left-1/2 transform -translate-x-1/2 px-4 py-2 bg-white/10 rounded-lg backdrop-blur-sm text-white text-sm text-center">
        拖拽移动 | 滚轮缩放 | +/- 缩放 | R 旋转 | 0 重置 | ESC 关闭{hasMultiple ? ' | ←/→ 切换' : ''}
      </div>

      {/* Instructions - mobile only */}
      <div className="sm:hidden absolute bottom-4 left-1/2 transform -translate-x-1/2 px-4 py-2 bg-white/10 rounded-lg backdrop-blur-sm text-white text-xs text-center">
        双指缩放 | 拖拽移动{hasMultiple ? ' | 左右滑动切换' : ''}
      </div>
    </div>
  );
}
