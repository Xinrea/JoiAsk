"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import Link from "next/link";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
import { getUsers, createUser, updateUser, deleteUser, User, UserRole } from "@/lib/api";
import { useAuth } from "@/lib/auth";

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleString("zh-CN");
}

export default function UsersPage() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return <p className="text-muted-foreground">加载中...</p>;
  }

  if (user?.role !== "super_admin") {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-semibold text-foreground">无权访问账号管理</h1>
        <p className="text-sm text-muted-foreground">
          仅超级管理员可管理后台账号。你仍可使用其他业务和配置功能。
        </p>
        <Button asChild variant="outline">
          <Link href="/dashboard/profile">编辑个人资料</Link>
        </Button>
      </div>
    );
  }

  return <UserManagement />;
}

function UserManagement() {
  const { user: currentUser, checkAuth } = useAuth();
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
  const [formData, setFormData] = useState<{ username: string; password: string; role: UserRole }>({
    username: "",
    password: "",
    role: "admin",
  });
  const [error, setError] = useState("");
  const [loadError, setLoadError] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  const loadUsers = useCallback(async () => {
    setIsLoading(true);
    setLoadError("");
    try {
      const res = await getUsers();
      if (res.code === 200 && res.data) {
        setUsers(res.data);
      } else {
        setLoadError(res.message || "加载账号失败，请重试");
      }
    } catch {
      setLoadError("加载账号失败，请检查网络连接后重试");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  const handleSubmit = async () => {
    if (isSaving) return;
    setError("");
    if (!formData.username.trim()) {
      setError("用户名不能为空");
      return;
    }
    if (dialog.mode === "add" && !formData.password.trim()) {
      setError("密码不能为空");
      return;
    }

    setIsSaving(true);
    try {
      const data = {
        username: formData.username,
        password: formData.password,
        role: formData.role,
      };

      if (dialog.mode === "add") {
        const res = await createUser(data);
        if (res.code === 200) {
          setDialog({ open: false, mode: "add", user: null });
          setFormData({ username: "", password: "", role: "admin" });
          loadUsers();
        } else {
          setError(res.message || "添加失败");
        }
      } else if (dialog.user) {
        const res = await updateUser(dialog.user.id, data);
        if (res.code === 200) {
          setDialog({ open: false, mode: "add", user: null });
          setFormData({ username: "", password: "", role: "admin" });
          if (dialog.user.id === currentUser?.id) {
            const authenticated = await checkAuth();
            if (!authenticated || data.role !== "super_admin") return;
          }
          loadUsers();
        } else {
          setError(res.message || "修改失败");
        }
      }
    } catch {
      setError(dialog.mode === "add" ? "添加失败" : "修改失败");
    } finally {
      setIsSaving(false);
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
    setFormData({ username: "", password: "", role: "admin" });
    setDialog({ open: true, mode: "add", user: null });
    setError("");
  };

  const openEditDialog = (user: User) => {
    setFormData({ username: user.username, password: "", role: user.role });
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
          <p className="mt-1 text-sm text-muted-foreground">
            两种角色拥有相同的业务和配置权限，仅超级管理员可管理后台账号和角色。
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            不能删除当前登录账号，也不能删除或降级最后一位超级管理员。
          </p>
        </div>
        <Button onClick={openAddDialog}>
          <Plus className="h-4 w-4 mr-2" />
          添加用户
        </Button>
      </header>

      {loadError && (
        <div role="alert" className="flex items-center gap-3 text-sm text-destructive">
          {loadError}
          <Button variant="outline" size="sm" onClick={loadUsers}>重试</Button>
        </div>
      )}
      <section>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>用户名</TableHead>
              <TableHead>角色</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead>更新时间</TableHead>
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
            ) : users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8">
                  暂无数据
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>{user.id}</TableCell>
                  <TableCell className="font-medium">{user.username}</TableCell>
                  <TableCell>
                    {user.role === "super_admin" ? "超级管理员" : "管理员"}
                  </TableCell>
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
                        aria-label={`编辑 ${user.username}`}
                        onClick={() => openEditDialog(user)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={!currentUser || user.id === currentUser.id}
                        title={user.id === currentUser?.id ? "不能删除当前登录账号" : "删除账号"}
                        aria-label={user.id === currentUser?.id ? "不能删除当前登录账号" : `删除 ${user.username}`}
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
                  dialog.mode === "edit" ? "留空则不修改密码" : "请输入密码"
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="role">角色</Label>
              <Select
                value={formData.role}
                onValueChange={(role: UserRole) => setFormData({ ...formData, role })}
              >
                <SelectTrigger id="role" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="admin">管理员</SelectItem>
                  <SelectItem value="super_admin">超级管理员</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <p className="text-sm text-muted-foreground">
              修改密码或角色会使该账号的所有登录失效，需重新登录；仅修改用户名不会。
            </p>
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
            <Button onClick={handleSubmit} disabled={isSaving}>
              {isSaving ? "保存中..." : dialog.mode === "add" ? "添加" : "保存"}
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
