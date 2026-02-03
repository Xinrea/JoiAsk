'use client';

import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PostCard } from '@/components/post-card';
import { GoToTop } from '@/components/go-to-top';
import { getQuestions, getTags, getInfo, Tag, Question } from '@/lib/api';
import { useWebSocket } from '@/hooks/useWebSocket';

export default function SearchPage() {
  const [tags, setTags] = useState<Tag[]>([]);
  const [selectedTag, setSelectedTag] = useState<string>('all');
  const [searchValue, setSearchValue] = useState('');
  const [results, setResults] = useState<Question[]>([]);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);

  const { emojiUpdates, archiveUpdates, handleCardCursor, handleCardCursorLeave, getRemoteCursors } = useWebSocket();

  useEffect(() => {
    getTags().then((res) => {
      if (res.code === 200 && res.data) {
        setTags(res.data.reverse());
      }
    });

    getInfo().then((res) => {
      setIsLoggedIn(res.code === 200);
    });
  }, []);

  const handleSearch = async () => {
    if (!searchValue.trim()) return;

    try {
      const res = await getQuestions({
        page: 0,
        size: 20,
        search: searchValue,
        publish: true,
        tag_id: selectedTag && selectedTag !== 'all' ? parseInt(selectedTag) : undefined,
      });

      if (res.code === 200) {
        setResults(res.data.questions || []);
      }
    } catch (error) {
      console.error('Search failed:', error);
    }
    setHasSearched(true);
  };

  return (
    <div className="flex flex-col items-center">
      {/* Title */}
      <h1 className="text-4xl text-center pt-16 font-bold text-primary">提问搜索</h1>
      <p className="text-center text-primary mt-2">最多显示 20 条搜索结果</p>

      {/* Search Form */}
      <div className="flex justify-center mt-16 p-2 gap-2">
        <Select value={selectedTag} onValueChange={setSelectedTag}>
          <SelectTrigger className="w-[140px] fabric-pill border-none text-primary text-xs">
            <SelectValue placeholder="所有话题" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">所有话题</SelectItem>
            {tags.map((tag) => (
              <SelectItem key={tag.id} value={tag.id.toString()}>
                #{tag.tag_name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input
          type="text"
          className="w-1/3 min-w-[200px] rounded-l-lg"
          maxLength={50}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          placeholder="输入搜索内容..."
        />
        <Button
          onClick={handleSearch}
          className="rounded-l-none"
        >
          搜索
        </Button>
      </div>

      {/* Results */}
      {hasSearched && results.length === 0 && (
        <div className="text-primary mt-16">没有找到相关的提问(๑´• .̫ •ू`๑)</div>
      )}

      {results.map((question) => (
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

      <GoToTop />
    </div>
  );
}
