"use client";

import { useEffect, useState } from "react";
import { Loader2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { requestEmailBinding } from "@/lib/api";
import { useAccountAuth } from "@/lib/account-auth";

export function EmailBindingDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { user, emailEnabled } = useAccountAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (open) {
      setEmail("");
      setPassword("");
      setError("");
      setMessage("");
    }
  }, [open]);

  if (!open || !user || user.email || !emailEnabled) return null;

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const response = await requestEmailBinding(email, password);
      if (response.code === 200) {
        setMessage(response.data.message);
        setPassword("");
      } else setError(response.message || "发送失败");
    } catch {
      setError("发送失败，请检查网络连接");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-[1200] flex items-center justify-center bg-black/45 px-4 py-8"
      onMouseDown={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="email-binding-title"
        className="w-full max-w-[420px] rounded-md border-2 border-dashed border-[var(--color-fabric-stitch)] bg-card p-5 text-foreground shadow-xl"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="mb-5 flex items-center justify-between">
          <h2 id="email-binding-title" className="font-semibold">
            绑定邮箱
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="关闭"
            className="flex h-9 w-9 items-center justify-center rounded hover:bg-secondary"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <form onSubmit={submit} className="space-y-4">
          <p className="text-sm text-muted-foreground">
            绑定后可通过邮箱找回密码。打开确认邮件中的链接后绑定才会生效。
          </p>
          <div>
            <label htmlFor="binding-email" className="mb-1.5 block text-sm">
              邮箱地址
            </label>
            <Input
              id="binding-email"
              type="email"
              autoComplete="email"
              maxLength={254}
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </div>
          <div>
            <label htmlFor="binding-password" className="mb-1.5 block text-sm">
              当前登录密码
            </label>
            <Input
              id="binding-password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </div>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          {message && (
            <p role="status" className="text-sm text-primary">
              {message}
            </p>
          )}
          <Button type="submit" className="w-full" disabled={busy}>
            {busy ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              "发送确认邮件"
            )}
          </Button>
        </form>
      </div>
    </div>
  );
}
