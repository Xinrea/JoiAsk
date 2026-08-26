"use client";

import { useEffect, useState } from "react";
import { CheckCircle2, Loader2, UserRound, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  AccountUser,
  VerificationStatus,
  getAccountVerification,
  loginAccount,
  registerAccount,
  startAccountVerification,
} from "@/lib/api";

interface AccountDialogProps {
  open: boolean;
  initialMode: "login" | "register";
  onClose: () => void;
  onAuthenticated: (user: AccountUser) => void;
}

export function AccountDialog({
  open,
  initialMode,
  onClose,
  onAuthenticated,
}: AccountDialogProps) {
  const [mode, setMode] = useState<"login" | "register">(initialMode);
  const [username, setUsername] = useState("");
  const [loginUsername, setLoginUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [verification, setVerification] = useState<VerificationStatus | null>(
    null,
  );
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [remaining, setRemaining] = useState(0);

  useEffect(() => {
    if (open) {
      setMode(initialMode);
      setUsername("");
      setLoginUsername("");
      setPassword("");
      setConfirmPassword("");
      setVerification(null);
      setRemaining(0);
      setError("");
    }
  }, [open, initialMode]);

  useEffect(() => {
    if (!open || mode !== "register" || verification) return;
    getAccountVerification()
      .then((response) => {
        if (
          response.code === 200 &&
          ["pending", "verified"].includes(response.data.status)
        ) {
          setVerification(response.data);
        }
      })
      .catch(() => undefined);
  }, [open, mode, verification]);

  useEffect(() => {
    if (!open || verification?.status !== "pending") return;
    const poll = window.setInterval(async () => {
      try {
        const response = await getAccountVerification();
        if (response.code === 200) setVerification(response.data);
      } catch {
        setError("暂时无法查询验证状态，系统会继续重试");
      }
    }, 5000);
    return () => window.clearInterval(poll);
  }, [open, verification?.status]);

  useEffect(() => {
    if (!verification) return;
    const tick = () => {
      const end =
        verification.status === "verified"
          ? verification.confirmation_until
          : verification.expires_at;
      setRemaining(
        end
          ? Math.max(
              0,
              Math.ceil((new Date(end).getTime() - Date.now()) / 1000),
            )
          : 0,
      );
    };
    tick();
    const timer = window.setInterval(tick, 1000);
    return () => window.clearInterval(timer);
  }, [verification]);

  if (!open) return null;

  const handleLogin = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const response = await loginAccount(loginUsername, password);
      if (response.code === 200) onAuthenticated(response.data);
      else setError(response.message || "登录失败");
    } catch {
      setError("登录失败，请检查网络连接");
    } finally {
      setBusy(false);
    }
  };

  const startVerification = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const response = await startAccountVerification(username);
      if (response.code === 200) setVerification(response.data);
      else setError(response.message || "无法开始验证");
    } catch {
      setError("无法开始验证，请稍后重试");
    } finally {
      setBusy(false);
    }
  };

  const handleRegister = async (event: React.FormEvent) => {
    event.preventDefault();
    if (password !== confirmPassword) {
      setError("两次输入的密码不一致");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const response = await registerAccount(loginUsername, password);
      if (response.code === 200) onAuthenticated(response.data);
      else setError(response.message || "注册失败");
    } catch {
      setError("注册失败，请稍后重试");
    } finally {
      setBusy(false);
    }
  };

  const resetRegister = () => {
    setVerification(null);
    setPassword("");
    setConfirmPassword("");
    setError("");
  };

  return (
    <div
      className="fixed inset-0 z-[1200] flex items-center justify-center bg-black/45 px-4 py-8"
      onMouseDown={onClose}
    >
      <div
        className="w-full max-w-[420px] rounded-md border-2 border-dashed border-[var(--fabric-stitch)] bg-card p-5 text-foreground shadow-xl"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="mb-5 flex items-center justify-between">
          <div className="flex rounded-md bg-secondary p-1">
            <button
              className={`rounded px-4 py-1.5 text-sm ${mode === "login" ? "bg-card shadow-sm" : "text-muted-foreground"}`}
              onClick={() => {
                setMode("login");
                resetRegister();
              }}
            >
              登录
            </button>
            <button
              className={`rounded px-4 py-1.5 text-sm ${mode === "register" ? "bg-card shadow-sm" : "text-muted-foreground"}`}
              onClick={() => {
                setMode("register");
                setError("");
              }}
            >
              注册
            </button>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="flex h-9 w-9 items-center justify-center rounded hover:bg-secondary"
            aria-label="关闭"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {mode === "login" ? (
          <form className="space-y-4" onSubmit={handleLogin}>
            <div>
              <label className="mb-1.5 block text-sm">登录名</label>
              <Input
                value={loginUsername}
                onChange={(e) => setLoginUsername(e.target.value)}
                autoComplete="username"
                required
              />
            </div>
            <div>
              <label className="mb-1.5 block text-sm">密码</label>
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button className="w-full" type="submit" disabled={busy}>
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "登录"}
            </Button>
          </form>
        ) : !verification ? (
          <form className="space-y-4" onSubmit={startVerification}>
            <div>
              <label className="mb-1.5 block text-sm">你的 B 站 UID</label>
              <Input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                inputMode="numeric"
                required
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button className="w-full" type="submit" disabled={busy}>
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "开始验证"}
            </Button>
          </form>
        ) : verification.status === "pending" ? (
          <div className="space-y-4">
            <div className="rounded-md bg-secondary p-4 text-sm leading-6">
              请在 3 分钟内关注 B 站账号{" "}
              <a
                href={`https://space.bilibili.com/${verification.target_uid}`}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-primary underline underline-offset-2"
              >
                {verification.target_uid}
              </a>
              。如果已经关注，请先取关再重新关注。系统预计会在两分钟内完成检查；验证完成后再取关不会影响你的账号状态。
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                等待系统确认（预计 2 分钟内）
              </span>
              <span>{remaining} 秒</span>
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            {remaining === 0 && (
              <Button
                className="w-full"
                variant="outline"
                onClick={resetRegister}
              >
                重新发起验证
              </Button>
            )}
          </div>
        ) : verification.status === "verified" ? (
          <form className="space-y-4" onSubmit={handleRegister}>
            <div className="rounded-md bg-secondary p-3">
              <div className="flex items-center gap-1.5 font-medium">
                <CheckCircle2 className="h-4 w-4 text-green-600" />
                {verification.bilibili_name}
              </div>
              <div className="mt-1 text-sm text-muted-foreground">
                UID{" "}
                <a
                  href={`https://space.bilibili.com/${verification.bilibili_uid}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary underline underline-offset-2"
                >
                  {verification.bilibili_uid}
                </a>
              </div>
            </div>
            <div>
              <label className="mb-1.5 block text-sm">设置登录名</label>
              <Input
                value={loginUsername}
                onChange={(e) => setLoginUsername(e.target.value)}
                minLength={2}
                maxLength={32}
                autoComplete="username"
                required
                placeholder="请输入登录名"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-sm">设置密码</label>
              <Input
                type="password"
                minLength={8}
                maxLength={72}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <div>
              <label className="mb-1.5 block text-sm">确认密码</label>
              <Input
                type="password"
                minLength={8}
                maxLength={72}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
              />
            </div>
            <p className="text-xs text-muted-foreground">
              请在 {Math.ceil(remaining / 60)} 分钟内完成注册
            </p>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button className="w-full" type="submit" disabled={busy}>
              {busy ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                "确认并创建账号"
              )}
            </Button>
          </form>
        ) : (
          <div className="space-y-4 text-center">
            <UserRound className="mx-auto h-8 w-8 text-muted-foreground" />
            <p>验证已过期，请重新发起。</p>
            <Button className="w-full" onClick={resetRegister}>
              重新验证
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
