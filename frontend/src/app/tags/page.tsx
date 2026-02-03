'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { getTags, Tag } from '@/lib/api';
import { GoToTop } from '@/components/go-to-top';

export default function TagsPage() {
  const [tags, setTags] = useState<Tag[]>([]);

  useEffect(() => {
    getTags().then((res) => {
      if (res.code === 200 && res.data) {
        setTags(res.data);
      }
    });
  }, []);

  return (
    <div className="flex flex-col items-center py-5">
      <div className="w-[600px] max-w-[95%] flex flex-col rounded-md fabric-form transition-all fabric-shadow-hover">
        {tags.map((tag, index) => (
          <div
            key={tag.id}
            className={`${index !== tags.length - 1 ? 'border-b-2 border-dashed border-[var(--fabric-stitch)]' : ''}`}
          >
            <div className="flex items-center justify-between px-5 py-2.5">
              <Link
                href={`/tags/${tag.id}?name=${encodeURIComponent(tag.tag_name)}`}
                className="text-primary hover:underline transition-colors duration-200"
              >
                #{tag.tag_name}
              </Link>
              <span className="text-muted-foreground text-xs ml-2">
                投稿数：{tag.question_count}
              </span>
            </div>
            <div className="text-muted-foreground text-sm px-5 pb-2.5">
              {tag.description}
            </div>
          </div>
        ))}
      </div>

      <GoToTop />
    </div>
  );
}
