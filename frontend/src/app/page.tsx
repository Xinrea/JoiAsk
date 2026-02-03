'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { Button } from '@/components/ui/button';
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
import { FileUpload } from '@/components/file-upload';
import { InputEmojiPicker } from '@/components/input-emoji-picker';
import { GoToTop } from '@/components/go-to-top';
import { getQuestions, getTags, getConfig, getInfo, createQuestion, Tag, Question } from '@/lib/api';
import { useWebSocket } from '@/hooks/useWebSocket';

export default function HomePage() {
  const [tags, setTags] = useState<Tag[]>([]);
  const [selectedTag, setSelectedTag] = useState<string>('');
  const [content, setContent] = useState('');
  const [isRainbow, setIsRainbow] = useState(false);
  const [showImageUpload, setShowImageUpload] = useState(false);
  const [files, setFiles] = useState<File[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const [announcement, setAnnouncement] = useState('');
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  // Questions state
  const [questions, setQuestions] = useState<Question[]>([]);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [hideArchive, setHideArchive] = useState(false);
  const [sortOrder, setSortOrder] = useState<'desc' | 'asc'>('desc');

  const { emojiUpdates, archiveUpdates, handleCardCursor, handleCardCursorLeave, getRemoteCursors } = useWebSocket();

  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Load initial data
  useEffect(() => {
    const savedContent = localStorage.getItem('ask_content');
    if (savedContent) setContent(savedContent);

    getTags().then((res) => {
      if (res.code === 200 && res.data) {
        const sorted = [res.data[0], ...res.data.slice(1).reverse()];
        setTags(sorted);
      }
    });

    getConfig().then((res) => {
      if (res.code === 200) {
        setAnnouncement(res.data.announcement || '');
      }
    });

    getInfo().then((res) => {
      setIsLoggedIn(res.code === 200);
    });
  }, []);

  // Load questions
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
  }, [page, sortOrder, hideArchive, isLoading]);

  // Initial load
  useEffect(() => {
    loadQuestions(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sortOrder, hideArchive]);

  // Infinite scroll
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

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setContent(e.target.value);
    localStorage.setItem('ask_content', e.target.value);
  };

  const handleEmojiInsert = (tag: string) => {
    const textarea = textareaRef.current;
    if (!textarea) {
      setContent((prev) => prev + tag);
      localStorage.setItem('ask_content', content + tag);
      return;
    }

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const newContent = content.slice(0, start) + tag + content.slice(end);
    setContent(newContent);
    localStorage.setItem('ask_content', newContent);

    // Restore cursor position after the inserted emoji
    setTimeout(() => {
      textarea.focus();
      textarea.setSelectionRange(start + tag.length, start + tag.length);
    }, 0);
  };

  const handleSubmit = async () => {
    const trimmedContent = content.trim().replace(/｛/g, '{').replace(/｝/g, '}');
    if (!trimmedContent) {
      alert('请输入有效的提问内容');
      return;
    }
    if (!selectedTag) {
      alert('请选择话题');
      return;
    }

    setIsSubmitting(true);
    const formData = new FormData();
    formData.append('tag_id', selectedTag);
    formData.append('content', trimmedContent);
    if (isRainbow) formData.append('rainbow', 'true');
    files.forEach((file) => formData.append('files[]', file));

    try {
      const res = await createQuestion(formData);
      if (res.ok) {
        const data = await res.json();
        if (data.code === 200) {
          setSubmitSuccess(true);
          setContent('');
          localStorage.setItem('ask_content', '');
          setSelectedTag('');
          setIsRainbow(false);
          setShowImageUpload(false);
          setFiles([]);
          setTimeout(() => setSubmitSuccess(false), 4000);
        } else {
          alert(data.message);
        }
      } else if (res.status === 413) {
        alert('图片太大，请缩小图片体积或者分开投稿');
      } else {
        alert('投稿出现错误');
      }
    } catch (error) {
      console.error('Submit error:', error);
      alert('投稿失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // Preview card data
  const previewCard: Question | null = content && selectedTag ? {
    id: 0,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    tag_id: parseInt(selectedTag),
    tag: tags.find((t) => t.id === parseInt(selectedTag)) || { id: 0, tag_name: '', description: '', question_count: 0, created_at: '', updated_at: '' },
    content: '[卡片预览]\n' + content,
    images: files.map((f) => URL.createObjectURL(f)).join(';'),
    images_num: files.length,
    is_hide: false,
    is_rainbow: isRainbow,
    is_archive: false,
    is_publish: false,
    emojis: '[]',
    likes: 0,
  } : null;

  return (
    <div className="flex flex-col w-full items-center">
      {/* Submit Form */}
      <div className="ask max-w-[600px] w-5/6 relative my-8 p-5 flex flex-col rounded-md fabric-form transition-all duration-220">
        {/* Tag Select */}
        <div className="mb-2.5">
          <Select value={selectedTag} onValueChange={setSelectedTag}>
            <SelectTrigger className="w-[140px] fabric-pill border-none text-primary text-xs">
              <SelectValue placeholder="选择话题" />
            </SelectTrigger>
            <SelectContent>
              {tags.map((tag) => (
                <SelectItem key={tag.id} value={tag.id.toString()}>
                  #{tag.tag_name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {selectedTag && (
            <pre className="text-primary text-sm mt-2 whitespace-pre-wrap">
              {tags.find((t) => t.id === parseInt(selectedTag))?.description}
            </pre>
          )}
        </div>

        {/* Content */}
        <div className={`relative ${isRainbow ? 'rainbow-bg' : ''}`}>
          <textarea
            ref={textareaRef}
            className={`w-full min-h-[128px] p-4 text-foreground bg-secondary/50 border-2 border-dashed border-[var(--fabric-stitch)] rounded-lg resize-none focus:bg-card focus:border-primary focus:outline-none transition-all duration-220 ${
              isRainbow ? 'rainbow-bg' : ''
            }`}
            style={{ fontFamily: '宋体, Fangsong, STFangsong, sans-serif' }}
            maxLength={800}
            value={content}
            onChange={handleContentChange}
            placeholder="输入你的提问..."
          />
          <div className="absolute bottom-4 right-4">
            <InputEmojiPicker onSelect={handleEmojiInsert} />
          </div>
        </div>

        {/* Options */}
        <div className="flex justify-between my-4 text-sm">
          <div className="text-primary whitespace-pre-wrap">{announcement}</div>
          <div className="flex flex-col items-start text-primary">
            <Label className="flex items-center cursor-pointer mb-1">
              <Checkbox
                checked={isRainbow}
                onCheckedChange={(checked) => setIsRainbow(!!checked)}
                className="mr-2"
              />
              彩虹屁
            </Label>
            <Label className="flex items-center cursor-pointer">
              <Checkbox
                checked={showImageUpload}
                onCheckedChange={(checked) => setShowImageUpload(!!checked)}
                className="mr-2"
              />
              附图
            </Label>
          </div>
        </div>

        {/* File Upload */}
        {showImageUpload && (
          <div className="mb-4">
            <FileUpload files={files} onFilesChange={setFiles} maxFiles={6} />
          </div>
        )}

        {/* Submit Button */}
        <Button
          onClick={handleSubmit}
          disabled={isSubmitting}
        >
          {isSubmitting ? '上传中，请稍等' : '提交'}
        </Button>

        {/* Success Message */}
        {submitSuccess && (
          <div className="absolute inset-0 flex bg-card text-foreground text-center items-center justify-center z-10 rounded-md border-2 border-dashed border-[var(--fabric-stitch)]">
            提问已提交审核，内容将会在审核通过后放出
          </div>
        )}
      </div>

      {/* Preview Card */}
      {previewCard && <PostCard data={previewCard} isLoggedIn={false} />}

      {/* Filters */}
      <div className="max-w-[600px] w-5/6 flex justify-between items-center mb-4 p-3 rounded-md bg-card border-2 border-dashed border-[var(--fabric-stitch)]">
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

      <style jsx>{`
        .rainbow-bg {
          background: linear-gradient(
            130deg,
            rgba(220, 93, 231, 0.33) 0%,
            rgba(127, 239, 189, 0.33) 33%,
            rgba(255, 245, 137, 0.33) 66%,
            rgba(236, 11, 67, 0.33) 100%
          );
        }
      `}</style>

      <GoToTop />
    </div>
  );
}
