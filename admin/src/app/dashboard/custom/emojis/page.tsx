"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  createInputEmoji,
  deleteInputEmoji,
  getInputEmojis,
  InputEmoji,
  updateInputEmoji,
  uploadConfigAsset,
} from "@/lib/api";
import { useEmojiCatalog } from "@/lib/emoji-catalog";

type EmojiForm = { tag: string; url: string };

const emptyForm: EmojiForm = { tag: "", url: "" };

export default function InputEmojisPage() {
  const { refresh } = useEmojiCatalog();
  const [emojis, setEmojis] = useState<InputEmoji[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [addDialog, setAddDialog] = useState(false);
  const [editDialog, setEditDialog] = useState<{ open: boolean; emoji: InputEmoji | null }>({
    open: false,
    emoji: null,
  });
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; id: number }>({
    open: false,
    id: 0,
  });
  const [form, setForm] = useState<EmojiForm>(emptyForm);
  const [error, setError] = useState("");

  const loadEmojis = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await getInputEmojis();
      if (res.code === 200 && res.data) setEmojis(res.data);
      else setError(res.message || "加载失败");
    } catch {
      setError("加载失败");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadEmojis();
  }, [loadEmojis]);

  const resetForm = () => {
    setForm(emptyForm);
    setError("");
  };

  const openAddDialog = () => {
    resetForm();
    setAddDialog(true);
  };

  const openEditDialog = (emoji: InputEmoji) => {
    setForm({ tag: emoji.tag, url: emoji.url });
    setError("");
    setEditDialog({ open: true, emoji });
  };

  const validateForm = () => {
    if (!form.tag.trim()) {
      setError("标签不能为空");
      return false;
    }
    if (!form.url.trim()) {
      setError("图片地址不能为空");
      return false;
    }
    return true;
  };

  const handleUpload = async (file: File | undefined) => {
    if (!file) return;
    setUploading(true);
    setError("");
    try {
      const res = await uploadConfigAsset(file);
      if (res.code === 200 && res.data) setForm((current) => ({ ...current, url: res.data.url }));
      else setError(res.message || "图片上传失败");
    } catch {
      setError("图片上传失败");
    } finally {
      setUploading(false);
    }
  };

  const handleAdd = async () => {
    if (!validateForm()) return;
    setIsSaving(true);
    try {
      const res = await createInputEmoji({ tag: form.tag.trim(), url: form.url.trim() });
      if (res.code === 200) {
        setAddDialog(false);
        resetForm();
        await loadEmojis();
        await refresh();
      } else {
        setError(res.message || "添加失败");
      }
    } catch {
      setError("添加失败");
    } finally {
      setIsSaving(false);
    }
  };

  const handleEdit = async () => {
    if (!editDialog.emoji || !validateForm()) return;
    setIsSaving(true);
    try {
      const res = await updateInputEmoji(editDialog.emoji.id, {
        tag: form.tag.trim(),
        url: form.url.trim(),
      });
      if (res.code === 200) {
        setEditDialog({ open: false, emoji: null });
        resetForm();
        await loadEmojis();
        await refresh();
      } else {
        setError(res.message || "修改失败");
      }
    } catch {
      setError("修改失败");
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    try {
      const res = await deleteInputEmoji(deleteDialog.id);
      if (res.code === 200) {
        await loadEmojis();
        await refresh();
      }
      else alert(res.message || "删除失败");
    } catch {
      alert("删除失败");
    } finally {
      setDeleteDialog({ open: false, id: 0 });
    }
  };

  const formContent = (idPrefix: string) => (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-tag`}>标签</Label>
        <Input
          id={`${idPrefix}-tag`}
          value={form.tag}
          onChange={(event) => setForm({ ...form, tag: event.target.value })}
          placeholder="[轴伊Joi收藏集动态表情包_跑了]"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-url`}>图片地址</Label>
        <div className="flex min-w-0 items-center gap-2">
          <Input
            id={`${idPrefix}-url`}
            type="url"
            className="min-w-0 flex-1"
            value={form.url}
            onChange={(event) => setForm({ ...form, url: event.target.value })}
            placeholder="/upload-img/emoji.webp"
          />
          <Input
            id={`${idPrefix}-file`}
            type="file"
            className="w-auto max-w-[45%] shrink-0"
            accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
            disabled={uploading}
            onChange={(event) => {
              void handleUpload(event.target.files?.[0]);
              event.currentTarget.value = "";
            }}
          />
        </div>
        {uploading && <p className="text-sm text-muted-foreground">图片上传中...</p>}
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
    </div>
  );

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <Button asChild variant="ghost" size="sm" className="mb-2 -ml-3">
            <Link href="/dashboard/custom">
              <ArrowLeft className="h-4 w-4" />
              返回自定义
            </Link>
          </Button>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">表情包管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            管理前台提问编辑器中的可插入表情包
          </p>
        </div>
        <Button onClick={openAddDialog}>
          <Plus className="h-4 w-4" />
          添加表情包
        </Button>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>表情包列表</CardTitle>
          <CardDescription>保存后，前台刷新页面或重新打开表情选择器即可获取最新列表。</CardDescription>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="p-3">预览</th>
                <th className="p-3">标签</th>
                <th className="p-3">图片地址</th>
                <th className="p-3">操作</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr><td colSpan={4} className="p-8 text-center">加载中...</td></tr>
              ) : emojis.length === 0 ? (
                <tr><td colSpan={4} className="p-8 text-center">暂无数据</td></tr>
              ) : emojis.map((emoji) => (
                <tr key={emoji.id} className="border-b">
                  <td className="p-3">
                    <img src={emoji.url} alt={emoji.tag} className="h-12 w-12 object-contain" />
                  </td>
                  <td className="max-w-[360px] p-3 font-medium break-all">{emoji.tag}</td>
                  <td className="max-w-[300px] truncate p-3 text-muted-foreground">{emoji.url}</td>
                  <td className="p-3">
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="sm" onClick={() => openEditDialog(emoji)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setDeleteDialog({ open: true, id: emoji.id })}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>

      <Dialog open={addDialog} onOpenChange={setAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加表情包</DialogTitle>
            <DialogDescription>填写标签和图片地址，也可以直接上传图片。</DialogDescription>
          </DialogHeader>
          {formContent("add-emoji")}
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddDialog(false)}>取消</Button>
            <Button onClick={handleAdd} disabled={isSaving || uploading}>
              {isSaving ? "添加中..." : "添加"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={editDialog.open}
        onOpenChange={(open) => setEditDialog({ ...editDialog, open })}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑表情包</DialogTitle>
            <DialogDescription>修改标签或图片地址。</DialogDescription>
          </DialogHeader>
          {formContent("edit-emoji")}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDialog({ open: false, emoji: null })}>取消</Button>
            <Button onClick={handleEdit} disabled={isSaving || uploading}>
              {isSaving ? "保存中..." : "保存"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={deleteDialog.open}
        onOpenChange={(open) => setDeleteDialog({ ...deleteDialog, open })}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>删除表情包</DialogTitle>
            <DialogDescription>确定要删除这个表情包吗？删除后前台将不再显示它。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialog({ open: false, id: 0 })}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
