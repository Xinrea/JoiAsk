"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { updateUser, User } from "@/lib/api";
import { useAuth } from "@/lib/auth";

export default function ProfilePage() {
  const { user } = useAuth();

  if (!user) return null;

  return <ProfileForm key={user.id} user={user} />;
}

function ProfileForm({ user }: { user: User }) {
  const { checkAuth } = useAuth();
  const [username, setUsername] = useState(user.username);
  const [password, setPassword] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (isSaving) return;
    setError("");
    setMessage("");
    if (!username.trim()) {
      setError("用户名不能为空");
      return;
    }

    setIsSaving(true);
    try {
      // Self-service updates must never include a role, for either role.
      const res = await updateUser(user.id, { username, password });
      if (res.code !== 200) {
        setError(res.message || "保存失败");
        return;
      }
      setPassword("");
      // Password changes revoke this session. The dashboard layout returns
      // to login when checkAuth clears it; username-only edits keep it alive.
      if (await checkAuth()) {
        setMessage("个人资料已保存");
      }
    } catch {
      setError("保存失败，请检查网络连接");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="max-w-lg space-y-8">
      <header>
        <h1 className="text-2xl font-semibold text-foreground tracking-tight">个人资料</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          修改自己的用户名和密码。当前角色：{user.role === "super_admin" ? "超级管理员" : "管理员"}。
        </p>
      </header>
      <form onSubmit={handleSubmit} className="space-y-5">
        <div className="space-y-2">
          <Label htmlFor="profile-username">用户名</Label>
          <Input
            id="profile-username"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            required
            disabled={isSaving}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="profile-password">新密码</Label>
          <Input
            id="profile-password"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="留空则不修改密码"
            disabled={isSaving}
            aria-describedby="profile-password-help"
          />
          <p id="profile-password-help" className="text-sm text-muted-foreground">
            修改密码后，所有登录都会失效，你将返回登录页。仅修改用户名不会退出登录。
          </p>
        </div>
        {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
        {message && <p role="status" className="text-sm text-muted-foreground">{message}</p>}
        <Button type="submit" disabled={isSaving}>
          {isSaving ? "保存中..." : "保存"}
        </Button>
      </form>
    </div>
  );
}
