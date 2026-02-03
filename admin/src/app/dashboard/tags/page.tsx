"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Plus, Pencil, Trash2 } from "lucide-react";
import { getTags, createTag, updateTag, deleteTag, Tag } from "@/lib/api";

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleString("zh-CN");
}

export default function TagsPage() {
  const [tags, setTags] = useState<Tag[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [addDialog, setAddDialog] = useState(false);
  const [editDialog, setEditDialog] = useState<{
    open: boolean;
    tag: Tag | null;
  }>({ open: false, tag: null });
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    id: number;
  }>({ open: false, id: 0 });
  const [formData, setFormData] = useState({ tag_name: "", description: "" });
  const [error, setError] = useState("");

  const loadTags = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await getTags();
      if (res.code === 200 && res.data) {
        setTags(res.data);
      }
    } catch (error) {
      console.error("Failed to load tags:", error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTags();
  }, [loadTags]);

  const handleAdd = async () => {
    setError("");
    if (!formData.tag_name.trim()) {
      setError("话题名不能为空");
      return;
    }
    try {
      const res = await createTag(formData);
      if (res.code === 200) {
        setAddDialog(false);
        setFormData({ tag_name: "", description: "" });
        loadTags();
      } else {
        setError(res.message || "添加失败");
      }
    } catch {
      setError("添加失败");
    }
  };

  const handleEdit = async () => {
    setError("");
    if (!editDialog.tag) return;
    if (!formData.tag_name.trim()) {
      setError("话题名不能为空");
      return;
    }
    try {
      const res = await updateTag(editDialog.tag.id, formData);
      if (res.code === 200) {
        setEditDialog({ open: false, tag: null });
        setFormData({ tag_name: "", description: "" });
        loadTags();
      } else {
        setError(res.message || "修改失败");
      }
    } catch {
      setError("修改失败");
    }
  };

  const handleDelete = async () => {
    try {
      const res = await deleteTag(deleteDialog.id);
      if (res.code === 200) {
        loadTags();
      } else {
        alert(res.message || "删除失败");
      }
    } catch {
      alert("删除失败");
    } finally {
      setDeleteDialog({ open: false, id: 0 });
    }
  };

  const openEditDialog = (tag: Tag) => {
    setFormData({ tag_name: tag.tag_name, description: tag.description });
    setEditDialog({ open: true, tag });
    setError("");
  };

  const openAddDialog = () => {
    setFormData({ tag_name: "", description: "" });
    setAddDialog(true);
    setError("");
  };

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-foreground tracking-tight">
            话题管理
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            创建与编辑话题分类
          </p>
        </div>
        <Button onClick={openAddDialog}>
          <Plus className="h-4 w-4 mr-2" />
          添加话题
        </Button>
      </header>

      <section>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>话题名</TableHead>
              <TableHead>描述</TableHead>
              <TableHead>投稿数量</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8">
                  加载中...
                </TableCell>
              </TableRow>
            ) : tags.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8">
                  暂无数据
                </TableCell>
              </TableRow>
            ) : (
              tags.map((tag) => (
                <TableRow key={tag.id}>
                  <TableCell>{tag.id}</TableCell>
                  <TableCell className="font-medium">{tag.tag_name}</TableCell>
                  <TableCell className="max-w-[300px] truncate">
                    {tag.description}
                  </TableCell>
                  <TableCell>{tag.question_count}</TableCell>
                  <TableCell className="text-sm">
                    {formatDate(tag.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openEditDialog(tag)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          setDeleteDialog({ open: true, id: tag.id })
                        }
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </section>

      {/* Add Dialog */}
      <Dialog open={addDialog} onOpenChange={setAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加话题</DialogTitle>
            <DialogDescription>创建一个新的话题分类</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="tag_name">话题名</Label>
              <Input
                id="tag_name"
                value={formData.tag_name}
                onChange={(e) =>
                  setFormData({ ...formData, tag_name: e.target.value })
                }
                placeholder="请输入话题名"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">描述</Label>
              <Input
                id="description"
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                placeholder="请输入描述"
              />
            </div>
            {error && <div className="text-red-500 text-sm">{error}</div>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddDialog(false)}>
              取消
            </Button>
            <Button onClick={handleAdd}>添加</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog
        open={editDialog.open}
        onOpenChange={(open) => setEditDialog({ ...editDialog, open })}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑话题</DialogTitle>
            <DialogDescription>修改话题信息</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit_tag_name">话题名</Label>
              <Input
                id="edit_tag_name"
                value={formData.tag_name}
                onChange={(e) =>
                  setFormData({ ...formData, tag_name: e.target.value })
                }
                placeholder="请输入话题名"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit_description">描述</Label>
              <Input
                id="edit_description"
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                placeholder="请输入描述"
              />
            </div>
            {error && <div className="text-red-500 text-sm">{error}</div>}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setEditDialog({ open: false, tag: null })}
            >
              取消
            </Button>
            <Button onClick={handleEdit}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog
        open={deleteDialog.open}
        onOpenChange={(open) => setDeleteDialog({ ...deleteDialog, open })}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>删除话题</DialogTitle>
            <DialogDescription>
              确定要删除这个话题吗？如果该话题下有投稿，将无法删除。
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
