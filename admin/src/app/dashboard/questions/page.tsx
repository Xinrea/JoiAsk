"use client";

import { useState, useEffect, useCallback, Fragment } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ChevronDown, ChevronUp, Trash2 } from "lucide-react";
import {
  getQuestions,
  getTags,
  updateQuestion,
  deleteQuestion,
  Question,
  Tag,
} from "@/lib/api";

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleString("zh-CN");
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

function renderContent(content: string) {
  let rendered = content.replace(/｛/g, "{").replace(/｝/g, "}");
  rendered = rendered.replace(/</g, "&lt;").replace(/>/g, "&gt;");
  rendered = rendered.replace(
    /\{\{([\s\S]*?)\}\}/g,
    '<span class="bg-black text-black hover:text-white transition-colors">$1</span>'
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

export default function QuestionsPage() {
  const [questions, setQuestions] = useState<Question[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [tagMap, setTagMap] = useState<Record<number, Tag>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [orderBy, setOrderBy] = useState("id");
  const [order, setOrder] = useState("desc");
  const [tagFilter, setTagFilter] = useState<string>("0");
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    id: number;
  }>({ open: false, id: 0 });
  const [isLoading, setIsLoading] = useState(false);

  const loadTags = useCallback(async () => {
    try {
      const res = await getTags();
      if (res.code === 200 && res.data) {
        setTags(res.data);
        const map: Record<number, Tag> = {};
        res.data.forEach((tag) => {
          map[tag.id] = tag;
        });
        setTagMap(map);
      }
    } catch (error) {
      console.error("Failed to load tags:", error);
    }
  }, []);

  const loadQuestions = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await getQuestions({
        page,
        page_size: pageSize,
        order_by: orderBy,
        order,
        tag_id: tagFilter !== "0" ? parseInt(tagFilter) : undefined,
      });
      if (res.code === 200) {
        setQuestions(res.data.questions || []);
        setTotal(res.data.total);
      }
    } catch (error) {
      console.error("Failed to load questions:", error);
    } finally {
      setIsLoading(false);
    }
  }, [page, pageSize, orderBy, order, tagFilter]);

  useEffect(() => {
    loadTags();
  }, [loadTags]);

  useEffect(() => {
    loadQuestions();
  }, [loadQuestions]);

  const handleSort = (key: string) => {
    if (orderBy === key) {
      setOrder(order === "asc" ? "desc" : "asc");
    } else {
      setOrderBy(key);
      setOrder("desc");
    }
  };

  const toggleRow = (id: number) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(id)) {
      newExpanded.delete(id);
    } else {
      newExpanded.add(id);
    }
    setExpandedRows(newExpanded);
  };

  const handleCheckboxChange = async (
    question: Question,
    field: "is_rainbow" | "is_archive" | "is_publish",
    value: boolean
  ) => {
    try {
      const res = await updateQuestion(question.id, {
        tag_id: question.tag_id,
        is_hide: question.is_hide,
        is_rainbow: field === "is_rainbow" ? value : question.is_rainbow,
        is_archive: field === "is_archive" ? value : question.is_archive,
        is_publish: field === "is_publish" ? value : question.is_publish,
      });
      if (res.code === 200) {
        loadQuestions();
      }
    } catch (error) {
      console.error("Failed to update question:", error);
    }
  };

  const handleTagChange = async (question: Question, newTagId: number) => {
    try {
      const res = await updateQuestion(question.id, {
        tag_id: newTagId,
        is_hide: question.is_hide,
        is_rainbow: question.is_rainbow,
        is_archive: question.is_archive,
        is_publish: question.is_publish,
      });
      if (res.code === 200) {
        loadQuestions();
      }
    } catch (error) {
      console.error("Failed to update question:", error);
    }
  };

  const handleDelete = async () => {
    try {
      const res = await deleteQuestion(deleteDialog.id);
      if (res.code === 200) {
        loadQuestions();
      }
    } catch (error) {
      console.error("Failed to delete question:", error);
    } finally {
      setDeleteDialog({ open: false, id: 0 });
    }
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold text-foreground tracking-tight">
          提问管理
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          筛选、排序与管理全部提问
        </p>
      </header>

      {/* 筛选与排序工具栏 */}
      <section className="rounded-xl border border-border bg-card p-4 shadow-[0_1px_0_rgba(255,255,255,0.5)_inset]">
        <div className="flex flex-wrap items-center gap-4 sm:gap-6">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground whitespace-nowrap">
              话题
            </span>
            <Select
              value={tagFilter}
              onValueChange={(v) => {
                setTagFilter(v);
                setPage(1);
              }}
            >
              <SelectTrigger className="w-[160px]">
                <SelectValue placeholder="选择话题" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">全部话题</SelectItem>
                {tags.map((tag) => (
                  <SelectItem key={tag.id} value={tag.id.toString()}>
                    {tag.tag_name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground whitespace-nowrap">
              排序
            </span>
            <Select
              value={orderBy}
              onValueChange={(v) => {
                setOrderBy(v);
                setPage(1);
              }}
            >
              <SelectTrigger className="w-[120px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="id">ID</SelectItem>
                <SelectItem value="created_at">时间</SelectItem>
                <SelectItem value="images_num">附图数</SelectItem>
                <SelectItem value="is_rainbow">彩虹</SelectItem>
                <SelectItem value="is_archive">归档</SelectItem>
                <SelectItem value="is_publish">公开</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={order}
              onValueChange={(v) => {
                setOrder(v);
                setPage(1);
              }}
            >
              <SelectTrigger className="w-[90px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="desc">降序</SelectItem>
                <SelectItem value="asc">升序</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground whitespace-nowrap">
              每页
            </span>
            <Select
              value={pageSize.toString()}
              onValueChange={(v) => {
                setPageSize(parseInt(v));
                setPage(1);
              }}
            >
              <SelectTrigger className="w-[80px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10">10</SelectItem>
                <SelectItem value="20">20</SelectItem>
                <SelectItem value="50">50</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <span className="text-sm text-muted-foreground ml-auto">
            共 {total} 条
          </span>
        </div>
      </section>

      <section>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[48px]"></TableHead>
              <TableHead className="w-[56px]">ID</TableHead>
              <TableHead className="w-[120px]">话题</TableHead>
              <TableHead className="min-w-[200px] max-w-[420px]">
                内容
              </TableHead>
              <TableHead className="w-[56px] text-center">附图</TableHead>
              <TableHead className="w-[56px] text-center">彩虹</TableHead>
              <TableHead className="w-[56px] text-center">归档</TableHead>
              <TableHead className="w-[56px] text-center">公开</TableHead>
              <TableHead className="w-[140px]">时间</TableHead>
              <TableHead className="w-[60px] text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={10} className="text-center py-12">
                  加载中...
                </TableCell>
              </TableRow>
            ) : questions.length === 0 ? (
              <TableRow>
                <TableCell colSpan={10} className="text-center py-12">
                  暂无数据
                </TableCell>
              </TableRow>
            ) : (
              questions.map((question) => (
                <Fragment key={question.id}>
                  <TableRow>
                    <TableCell className="whitespace-nowrap">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => toggleRow(question.id)}
                        title={
                          expandedRows.has(question.id) ? "收起" : "展开详情"
                        }
                      >
                        {expandedRows.has(question.id) ? (
                          <ChevronUp className="h-4 w-4" />
                        ) : (
                          <ChevronDown className="h-4 w-4" />
                        )}
                      </Button>
                    </TableCell>
                    <TableCell className="font-mono text-sm whitespace-nowrap">
                      {question.id}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      <Select
                        value={question.tag_id.toString()}
                        onValueChange={(v) =>
                          handleTagChange(question, parseInt(v))
                        }
                      >
                        <SelectTrigger className="h-8 w-[100px] text-xs">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {tags.map((tag) => (
                            <SelectItem
                              key={tag.id}
                              value={tag.id.toString()}
                            >
                              {tag.tag_name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell className="max-w-[420px]">
                      <span
                        className="line-clamp-2 text-sm leading-relaxed"
                        dangerouslySetInnerHTML={{
                          __html: renderContent(question.content.slice(0, 120)),
                        }}
                      />
                      {question.content.length > 120 && (
                        <span className="text-muted-foreground">…</span>
                      )}
                    </TableCell>
                    <TableCell className="text-center text-sm whitespace-nowrap">
                      {question.images_num}
                    </TableCell>
                    <TableCell className="text-center whitespace-nowrap">
                      <Checkbox
                        checked={question.is_rainbow}
                        onCheckedChange={(checked) =>
                          handleCheckboxChange(
                            question,
                            "is_rainbow",
                            !!checked
                          )
                        }
                      />
                    </TableCell>
                    <TableCell className="text-center whitespace-nowrap">
                      <Checkbox
                        checked={question.is_archive}
                        onCheckedChange={(checked) =>
                          handleCheckboxChange(
                            question,
                            "is_archive",
                            !!checked
                          )
                        }
                      />
                    </TableCell>
                    <TableCell className="text-center whitespace-nowrap">
                      <Checkbox
                        checked={question.is_publish}
                        onCheckedChange={(checked) =>
                          handleCheckboxChange(
                            question,
                            "is_publish",
                            !!checked
                          )
                        }
                      />
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {formatDate(question.created_at)}
                    </TableCell>
                    <TableCell className="text-right whitespace-nowrap">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-destructive hover:text-destructive"
                        onClick={() =>
                          setDeleteDialog({ open: true, id: question.id })
                        }
                        title="删除"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                  {expandedRows.has(question.id) && (
                    <TableRow key={`${question.id}-expanded`}>
                      <TableCell
                        colSpan={10}
                        className="bg-muted/30 p-0 align-top"
                      >
                        <div className="w-full max-w-full overflow-hidden">
                          <div className="grid grid-cols-1 gap-6 p-5 lg:grid-cols-3">
                            {/* 左侧：完整内容 + 附图 */}
                            <div className="space-y-5 lg:col-span-2 min-w-0">
                              <div className="min-w-0">
                                <h4 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                                  完整内容
                                </h4>
                                <div
                                  className="whitespace-pre-wrap break-words rounded-lg border border-border bg-card px-4 py-3 text-sm leading-relaxed overflow-hidden"
                                  dangerouslySetInnerHTML={{
                                    __html: renderContent(question.content),
                                  }}
                                />
                              </div>
                              {question.images_num > 0 && (
                                <div className="min-w-0">
                                  <h4 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                                    附图
                                  </h4>
                                  <div className="flex flex-wrap gap-3">
                                    {question.images
                                      .split(";")
                                      .filter(Boolean)
                                      .map((img, idx) => (
                                        <a
                                          key={idx}
                                          href={img}
                                          target="_blank"
                                          rel="noopener noreferrer"
                                          className="block overflow-hidden rounded-lg border border-border shadow-sm transition hover:shadow-md flex-shrink-0"
                                        >
                                          <img
                                            src={img}
                                            alt={`附图 ${idx + 1}`}
                                            className="h-32 w-32 object-cover"
                                          />
                                        </a>
                                      ))}
                                  </div>
                                </div>
                              )}
                            </div>
                            {/* 右侧：元数据 + 编辑 */}
                            <div className="space-y-5 lg:col-span-1 min-w-0">
                              <div>
                                <h4 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                                  元数据
                                </h4>
                                <ul className="space-y-1.5 rounded-lg border border-border bg-card px-4 py-3 text-sm">
                                  <li>ID: {question.id}</li>
                                  <li>点赞: {question.likes}</li>
                                  <li>图片: {question.images_num}</li>
                                  <li>时间: {formatDate(question.created_at)}</li>
                                </ul>
                              </div>
                              <div>
                                <h4 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                                  编辑
                                </h4>
                                <div className="space-y-3 rounded-lg border border-border bg-card p-4">
                                  <div className="flex items-center justify-between gap-2">
                                    <span className="text-sm">话题</span>
                                    <Select
                                      value={question.tag_id.toString()}
                                      onValueChange={(v) =>
                                        handleTagChange(question, parseInt(v))
                                      }
                                    >
                                      <SelectTrigger className="h-8 w-full max-w-[140px]">
                                        <SelectValue />
                                      </SelectTrigger>
                                      <SelectContent>
                                        {tags.map((tag) => (
                                          <SelectItem
                                            key={tag.id}
                                            value={tag.id.toString()}
                                          >
                                            {tag.tag_name}
                                          </SelectItem>
                                        ))}
                                      </SelectContent>
                                    </Select>
                                  </div>
                                  <div className="flex items-center justify-between gap-2">
                                    <span className="text-sm">彩虹</span>
                                    <Checkbox
                                      checked={question.is_rainbow}
                                      onCheckedChange={(c) =>
                                        handleCheckboxChange(
                                          question,
                                          "is_rainbow",
                                          !!c
                                        )
                                      }
                                    />
                                  </div>
                                  <div className="flex items-center justify-between gap-2">
                                    <span className="text-sm">归档</span>
                                    <Checkbox
                                      checked={question.is_archive}
                                      onCheckedChange={(c) =>
                                        handleCheckboxChange(
                                          question,
                                          "is_archive",
                                          !!c
                                        )
                                      }
                                    />
                                  </div>
                                  <div className="flex items-center justify-between gap-2">
                                    <span className="text-sm">公开</span>
                                    <Checkbox
                                      checked={question.is_publish}
                                      onCheckedChange={(c) =>
                                        handleCheckboxChange(
                                          question,
                                          "is_publish",
                                          !!c
                                        )
                                      }
                                    />
                                  </div>
                                  <Button
                                    variant="destructive"
                                    size="sm"
                                    className="w-full"
                                    onClick={() =>
                                      setDeleteDialog({
                                        open: true,
                                        id: question.id,
                                      })
                                    }
                                  >
                                    <Trash2 className="h-4 w-4 mr-2" />
                                    删除该提问
                                  </Button>
                                </div>
                              </div>
                            </div>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              ))
            )}
          </TableBody>
        </Table>
      </section>

      <section className="flex flex-wrap items-center justify-between gap-4">
        <span className="text-sm text-muted-foreground">
          第 {page} / {totalPages || 1} 页
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page === 1}
          >
            上一页
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            disabled={page >= totalPages}
          >
            下一页
          </Button>
        </div>
      </section>

      <Dialog
        open={deleteDialog.open}
        onOpenChange={(open) => setDeleteDialog({ ...deleteDialog, open })}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>删除提问</DialogTitle>
            <DialogDescription>
              确定要删除这条提问吗？此操作不可撤销。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialog({ open: false, id: 0 })}
            >
              取消
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
