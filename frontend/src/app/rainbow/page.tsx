'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { PostCard } from '@/components/post-card';
import { GoToTop } from '@/components/go-to-top';
import { getQuestions, getInfo, Question } from '@/lib/api';
import { useWebSocket } from '@/hooks/useWebSocket';

export default function RainbowPage() {
  const [questions, setQuestions] = useState<Question[]>([]);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
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
        rainbow: true,
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
  }, [page, isLoading]);

  useEffect(() => {
    loadQuestions(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
