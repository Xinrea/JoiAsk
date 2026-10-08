"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { confirmEmailBinding, resetAccountPassword } from "@/lib/api";
import { useAccountAuth } from "@/lib/account-auth";

export default function EmailLinkPage() {
  const { user, loading, emailEnabled, setUser, openAccountDialog } =
    useAccountAuth();
  const [link, setLink] = useState<{ token: string; purpose: string } | null>(
    null,
  );
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    const readLink = () => {
      const params = new URLSearchParams(window.location.hash.slice(1));
      setLink({
        token: params.get("token") || "",
        purpose: params.get("purpose") || "",
      });
      setPassword("");
      setConfirmation("");
      setError("");
      setMessage("");
    };
    readLink();
    window.addEventListener("hashchange", readLink);
    return () => window.removeEventListener("hashchange", readLink);
  }, []);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!link) return;
    if (link.purpose === "reset" && password !== confirmation) {
      setError("两次输入的密码不一致");
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (link.purpose === "bind") {
        const response = await confirmEmailBinding(link.token);
        if (response.code === 200) {
          setUser(response.data);
          setMessage("邮箱绑定成功，可用于找回密码。");
        } else setError(response.message || "绑定失败");
      } else {
        const response = await resetAccountPassword(link.token, password);
        if (response.code === 200) {
          setUser(null);
          setPassword("");
          setConfirmation("");
          setMessage(response.data.message);
        } else setError(response.message || "重置失败");
      }
    } catch {
      setError("请求失败，请检查网络连接");
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="mx-auto w-full max-w-lg px-4 py-12">
      <div className="rounded-md border-2 border-dashed border-[var(--color-fabric-stitch)] bg-card p-6 text-foreground shadow-sm">
        <h1 className="mb-5 text-xl font-semibold">
          {link?.purpose === "bind" ? "确认绑定邮箱" : "重新设置密码"}
        </h1>
        {!emailEnabled ? (
          <p className="text-sm text-muted-foreground">
            邮件功能未开启，请联系管理员。
          </p>
        ) : !link || loading ? (
          <p>加载中...</p>
        ) : !/^[a-f0-9]{64}$/.test(link.token) ||
          !["bind", "reset"].includes(link.purpose) ? (
          <p role="alert">邮件链接无效，请重新请求邮件。</p>
        ) : message ? (
          <div className="space-y-4">
            <p role="status">{message}</p>
            {link.purpose === "reset" && (
              <Button onClick={() => openAccountDialog("login")}>登录</Button>
            )}
          </div>
        ) : link.purpose === "bind" && !user ? (
          <div className="space-y-4">
            <p>请先登录发起邮箱绑定的账号，再确认绑定。</p>
            <Button onClick={() => openAccountDialog("login")}>登录</Button>
          </div>
        ) : (
          <form className="space-y-4" onSubmit={submit}>
            {link.purpose === "bind" ? (
              <p className="text-sm">
                将此邮件的收件邮箱绑定到当前账号 {user?.username}
                。请确认你正在登录发起绑定的账号。
              </p>
            ) : (
              <>
                <div>
                  <label
                    htmlFor="new-password"
                    className="mb-1.5 block text-sm"
                  >
                    新密码
                  </label>
                  <Input
                    id="new-password"
                    type="password"
                    autoComplete="new-password"
                    minLength={8}
                    maxLength={72}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label
                    htmlFor="confirm-password"
                    className="mb-1.5 block text-sm"
                  >
                    确认新密码
                  </label>
                  <Input
                    id="confirm-password"
                    type="password"
                    autoComplete="new-password"
                    minLength={8}
                    maxLength={72}
                    value={confirmation}
                    onChange={(e) => setConfirmation(e.target.value)}
                    required
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  密码至少 8 个字符；中文等字符会占用多个字节，总长度不能超过 72
                  字节。
                </p>
              </>
            )}
            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}
            <Button className="w-full" type="submit" disabled={busy}>
              {busy
                ? "处理中..."
                : link.purpose === "bind"
                  ? "确认绑定"
                  : "重置密码"}
            </Button>
          </form>
        )}
        <Link
          href="/"
          className="mt-5 inline-block text-sm text-primary hover:underline"
        >
          返回首页
        </Link>
      </div>
    </main>
  );
}
