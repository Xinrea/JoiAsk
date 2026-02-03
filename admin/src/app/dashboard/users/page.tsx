"use client";

import { useState, useEffect, useCallback } from "react";
import md5 from "md5";
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
import { getUsers, createUser, updateUser, deleteUser, User } from "@/lib/api";

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleString("zh-CN");
}

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [dialog, setDialog] = useState<{
    open: boolean;
    mode: "add" | "edit";
    user: User | null;
  }>({
    open: false,
    mode: "add",
    user: null,
  });
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    id: number;
  }>({ open: false, id: 0 });
  const [formData, setFormData] = useState({ username: "", password: "" });
  const [error, setError] = useState("");

  const loadUsers = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await getUsers();
      if (res.code === 200 && res.data) {
        setUsers(res.data);
      }
    } catch (error) {
      console.error("Failed to load users:", error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  const handleSubmit = async () => {
    setError("");
    if (!formData.username.trim()) {
      setError("用户名不能为空");
      return;
    }
    if (!formData.password.trim()) {
      setError("密码不能为空");
      return;
    }

    try {
      const data = {
        username: formData.username,
        password: md5(formData.password),
      };

      if (dialog.mode === "add") {
        const res = await createUser(data);
        if (res.code === 200) {
          setDialog({ open: false, mode: "add", user: null });
          setFormData({ username: "", password: "" });
          loadUsers();
        } else {
          setError(res.message || "添加失败");
        }
      } else if (dialog.user) {
        const res = await updateUser(dialog.user.id, data);
        if (res.code === 200) {
          setDialog({ open: false, mode: "add", user: null });
          setFormData({ username: "", password: "" });
          loadUsers();
        } else {
          setError(res.message || "修改失败");
        }
      }
    } catch {
      setError(dialog.mode === "add" ? "添加失败" : "修改失败");
    }
  };

  const handleDelete = async () => {
    try {
      const res = await deleteUser(deleteDialog.id);
      if (res.code === 200) {
        loadUsers();
      } else {
        alert(res.message || "删除失败");
      }
    } catch {
      alert("删除失败");
    } finally {
      setDeleteDialog({ open: false, id: 0 });
    }
  };

  const openAddDialog = () => {
    setFormData({ username: "", password: "" });
    setDialog({ open: true, mode: "add", user: null });
    setError("");
  };

  const openEditDialog = (user: User) => {
    setFormData({ username: user.username, password: "" });
    setDialog({ open: true, mode: "edit", user });
    setError("");
  };

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold text-foreground tracking-tight">
            账号管理
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">管理后台登录账号</p>
        </div>
        <Button onClick={openAddDialog}>
          <Plus className="h-4 w-4 mr-2" />
          添加用户
        </Button>
      </header>

      <section>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>用户名</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-8">
                  加载中...
                </TableCell>
              </TableRow>
            ) : users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-8">
                  暂无数据
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>{user.id}</TableCell>
                  <TableCell className="font-medium">{user.username}</TableCell>
                  <TableCell className="text-sm">
                    {formatDate(user.created_at)}
                  </TableCell>
                  <TableCell className="text-sm">
                    {formatDate(user.updated_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openEditDialog(user)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          setDeleteDialog({ open: true, id: user.id })
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

      {/* Add/Edit Dialog */}
      <Dialog
        open={dialog.open}
        onOpenChange={(open) => setDialog({ ...dialog, open })}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {dialog.mode === "add" ? "添加用户" : "编辑用户"}
            </DialogTitle>
            <DialogDescription>
              {dialog.mode === "add"
                ? "创建一个新的管理员账号"
                : "修改用户信息"}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                value={formData.username}
                onChange={(e) =>
                  setFormData({ ...formData, username: e.target.value })
                }
                placeholder="请输入用户名"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                type="password"
                value={formData.password}
                onChange={(e) =>
                  setFormData({ ...formData, password: e.target.value })
                }
                placeholder={
                  dialog.mode === "edit" ? "输入新密码" : "请输入密码"
                }
              />
            </div>
            {error && <div className="text-red-500 text-sm">{error}</div>}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() =>
                setDialog({ open: false, mode: "add", user: null })
              }
            >
              取消
            </Button>
            <Button onClick={handleSubmit}>
              {dialog.mode === "add" ? "添加" : "保存"}
            </Button>
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
            <DialogTitle>删除用户</DialogTitle>
            <DialogDescription>
              确定要删除这个用户吗？此操作不可撤销。
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
