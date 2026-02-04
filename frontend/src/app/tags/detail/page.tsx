'use client';

import { useState, useEffect, useCallback, useRef, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PostCard } from '@/components/post-card';
import { GoToTop } from '@/components/go-to-top';
import { getQuestions, getInfo, Question } from '@/lib/api';
import { useWebSocket } from '@/hooks/useWebSocket';

function TagDetailContent() {
  const searchParams = useSearchParams();
  const tagId = searchParams.get('id') || '';
  const tagName = searchParams.get('name') || '';

  const [questions, setQuestions] = useState<Question[]>([]);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [hideArchive, setHideArchive] = useState(false);
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  const { emojiUpdates, archiveUpdates, handleCardCursor, handleCardCursorLeave, getRemoteCursors } = useWebSocket();

  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    getInfo().then((res) => {
      setIsLoggedIn(res.code === 200);
    });
  }, []);

  const loadQuestions = useCallback(async (reset = false) => {
    if (isLoading) return;
    setIsLoading(true);

    const currentPage = reset ? 1 : page;
    try {
      const res = await getQuestions({
        page: currentPage,
        size: 5,
        publish: true,
        order_by: 'id',
        order: sortOrder,
        tag_id: parseInt(tagId),
        archive: hideArchive ? false : undefined,
      });

      if (res.code === 200) {
        const newQuestions = res.data.questions || [];
        if (reset) {
          setQuestions(newQuestions);
        } else {
          setQuestions((prev) => [...prev, ...newQuestions]);
        }
        setHasMore(newQuestions.length >= 5);
        setPage(currentPage + 1);
      }
    } catch (error) {
      console.error('Failed to load questions:', error);
    } finally {
      setIsLoading(false);
    }
  }, [page, sortOrder, hideArchive, isLoading, tagId]);

  useEffect(() => {
    setPage(1);
    setQuestions([]);
    loadQuestions(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sortOrder, hideArchive, tagId]);

  useEffect(() => {
    if (observerRef.current) {
      observerRef.current.disconnect();
    }

    observerRef.current = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !isLoading) {
          loadQuestions();
        }
      },
      { threshold: 0.1 }
    );

    if (loadMoreRef.current) {
      observerRef.current.observe(loadMoreRef.current);
    }

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, [hasMore, isLoading, loadQuestions]);

  return (
    <div className="flex flex-col items-center">
      {/* Tag Header */}
      <div
        className="my-12 px-2 py-1 text-center text-5xl leading-[50px] text-primary opacity-80"
        style={{
          fontFamily: 'system-ui',
          boxShadow: '0 0 0 3px var(--primary), 0 0 0 2px var(--primary) inset',
          border: '2px dashed transparent',
          borderRadius: '4px',
          width: '300px',
          transform: 'rotate(5deg)',
        }}
      >
        #{tagName}
      </div>

      {/* Filters */}
      <div className="max-w-[600px] w-5/6 flex justify-between items-center mb-4">
        <Label className="flex items-center cursor-pointer text-primary">
          <Checkbox
            checked={hideArchive}
            onCheckedChange={(checked) => {
              setHideArchive(!!checked);
              setPage(1);
              setQuestions([]);
            }}
            className="mr-2"
          />
          隐藏归档卡片
        </Label>
        <Select
          value={sortOrder}
          onValueChange={(v) => {
            setSortOrder(v as 'desc' | 'asc');
            setPage(1);
            setQuestions([]);
          }}
        >
          <SelectTrigger className="w-[120px] fabric-card border-2 border-dashed border-[var(--fabric-stitch)]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="desc">从新到旧</SelectItem>
            <SelectItem value="asc">从旧到新</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Questions List */}
      {questions.map((question) => (
        <PostCard
          key={question.id}
          data={question}
          isLoggedIn={isLoggedIn}
          emojiUpdates={emojiUpdates}
          archiveUpdates={archiveUpdates}
          remoteCursors={getRemoteCursors(question.id)}
          onCursorMove={(x, y) => handleCardCursor(question.id, x, y)}
          onCursorLeave={() => handleCardCursorLeave(question.id)}
        />
      ))}

      {/* Load More */}
      <div ref={loadMoreRef} className="py-4 text-primary text-center">
        {isLoading ? '加载中...' : hasMore ? '' : '已经到底啦( ´･･)ﾉ(._.`)'}
      </div>

      <GoToTop />
    </div>
  );
}

export default function TagDetailPage() {
  return (
    <Suspense fallback={<div className="flex justify-center py-12 text-primary">加载中...</div>}>
      <TagDetailContent />
    </Suspense>
  );
}
