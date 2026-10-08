"use client";

import Link from "next/link";
import { useState } from "react";
import { EmailBindingDialog } from "@/components/email-binding-dialog";
import { usePathname } from "next/navigation";
import { Github, LogOut, UserRound, Mail } from "lucide-react";
import { useAccountAuth } from "@/lib/account-auth";

const NAV_ROUTES = [
  { label: "提问", path: "/" },
  { label: "话题", path: "/tags" },
  { label: "彩虹屁", path: "/rainbow" },
  { label: "搜索", path: "/search" },
];

export function Navbar({ logoUrl }: { logoUrl: string }) {
  const pathname = usePathname();
  const [emailDialogOpen, setEmailDialogOpen] = useState(false);
  const { user, loading, emailEnabled, openAccountDialog, logout } =
    useAccountAuth();

  const isActive = (path: string) => {
    if (path === "/") return pathname === "/";
    return pathname.startsWith(path);
  };

  return (
    <>
      {emailEnabled && user && !user.email && (
        <EmailBindingDialog
          open={emailDialogOpen}
          onClose={() => setEmailDialogOpen(false)}
        />
      )}
      <div
        className="navbar fabric-nav text-[var(--color-nav-foreground)] h-14 px-2 flex items-center justify-between gap-1 z-[999]"
        style={{
          color:
            "var(--color-nav-foreground, var(--color-brand-foreground, #fff))",
        }}
      >
        <div className="h-full min-w-0 flex items-center overflow-x-auto">
          <Link href="/">
            <div
              className="cursor-pointer w-10 h-10 bg-contain bg-no-repeat mr-2.5 transition-transform duration-200 hover:scale-105"
              style={{ backgroundImage: `url(${logoUrl})` }}
            />
          </Link>
          {NAV_ROUTES.map(({ label, path }) => (
            <Link
              key={path}
              href={path}
              className={`shrink-0 whitespace-nowrap px-2 py-1.5 mx-0.5 rounded text-[var(--color-nav-foreground)] transition-all duration-200 sm:px-3 sm:mx-1 sm:text-base ${
                isActive(path)
                  ? "bg-white/20 border border-dashed border-white/30"
                  : "hover:bg-white/10"
              }`}
            >
              {label}
            </Link>
          ))}
        </div>
        <div className="flex min-w-0 items-center gap-2">
          {!loading && user ? (
            <>
              <div className="group relative hidden min-w-0 items-center gap-2 text-[var(--color-nav-foreground)] sm:flex">
                <div
                  className="h-8 w-8 shrink-0 rounded-full bg-cover bg-center"
                  style={{ backgroundImage: `url(${user.bilibili_avatar})` }}
                />
                <span
                  tabIndex={0}
                  aria-describedby="account-info-tooltip"
                  className="max-w-28 truncate rounded text-sm outline-offset-4"
                >
                  {user.bilibili_name || user.username}
                </span>
                <div
                  role="tooltip"
                  id="account-info-tooltip"
                  className="pointer-events-none invisible absolute right-0 top-full z-[1000] w-72 pt-2 opacity-0 transition-opacity group-hover:visible group-hover:opacity-100 group-focus-within:visible group-focus-within:opacity-100"
                >
                  <div className="space-y-2 rounded-md border border-border bg-card p-3 text-sm text-foreground shadow-lg">
                    <p className="break-all">登录名：{user.username}</p>
                    <p className="break-all text-muted-foreground">
                      B 站 UID：{user.bilibili_uid}
                    </p>
                    {emailEnabled && user.email && (
                      <p className="break-all">已绑定邮箱：{user.email}</p>
                    )}
                  </div>
                </div>
              </div>
              {emailEnabled && !user.email && (
                <button
                  type="button"
                  onClick={() => setEmailDialogOpen(true)}
                  className="relative flex h-9 w-9 shrink-0 items-center justify-center rounded text-[var(--color-nav-foreground)] hover:bg-white/10"
                  title="绑定邮箱以找回密码"
                  aria-label="绑定邮箱"
                >
                  <Mail className="h-4 w-4" />
                  <span
                    aria-hidden="true"
                    className="absolute right-1 top-1 h-2 w-2 rounded-full bg-red-500"
                  />
                </button>
              )}
              <button
                type="button"
                onClick={logout}
                className="flex h-9 w-9 items-center justify-center rounded text-[var(--color-nav-foreground)] hover:bg-white/10"
                title="退出登录"
                aria-label="退出登录"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </>
          ) : (
            <button
              type="button"
              disabled={loading}
              onClick={() => openAccountDialog("login")}
              className="flex h-9 shrink-0 items-center gap-1.5 rounded px-2 text-sm text-[var(--color-nav-foreground)] hover:bg-white/10 disabled:opacity-60 sm:px-3"
              title="登录 / 注册"
            >
              <UserRound className="h-4 w-4" />
              <span className="hidden whitespace-nowrap sm:inline">
                登录 / 注册
              </span>
            </button>
          )}
          <a
            href="https://github.com/Xinrea/JoiAsk"
            target="_blank"
            rel="noopener noreferrer"
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded text-[var(--color-nav-foreground)] transition-colors hover:bg-white/10"
            title="在 GitHub 上查看 JoiAsk"
            aria-label="在 GitHub 上查看 JoiAsk"
          >
            <Github className="h-5 w-5" />
          </a>
        </div>
      </div>
    </>
  );
}
