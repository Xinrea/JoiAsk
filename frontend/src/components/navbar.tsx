'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { LogOut, UserRound } from 'lucide-react';
import { useAccountAuth } from '@/lib/account-auth';

const NAV_ROUTES = [
  { label: '提问', path: '/' },
  { label: '话题', path: '/tags' },
  { label: '彩虹屁', path: '/rainbow' },
  { label: '搜索', path: '/search' },
];

export function Navbar() {
  const pathname = usePathname();
  const { user, loading, openAccountDialog, logout } = useAccountAuth();

  const isActive = (path: string) => {
    if (path === '/') return pathname === '/';
    return pathname.startsWith(path);
  };

  return (
    <div className="navbar fabric-nav h-14 px-2 flex items-center justify-between gap-1 z-[999]">
      <div className="h-full min-w-0 flex items-center overflow-x-auto">
        <Link href="/">
          <div
            className="cursor-pointer w-10 h-10 bg-contain bg-no-repeat mr-2.5 transition-transform duration-200 hover:scale-105"
            style={{ backgroundImage: 'url(/favicon.png)' }}
          />
        </Link>
        {NAV_ROUTES.map(({ label, path }) => (
          <Link
            key={path}
            href={path}
            className={`shrink-0 whitespace-nowrap px-2 py-1.5 mx-0.5 rounded text-xs text-primary-foreground transition-all duration-200 sm:px-3 sm:mx-1 sm:text-base ${
              isActive(path)
                ? 'bg-white/20 border border-dashed border-white/30'
                : 'hover:bg-white/10'
            }`}
          >
            {label}
          </Link>
        ))}
      </div>
      <div className="flex min-w-0 items-center gap-2">
        {!loading && user ? (
          <>
            <div className="hidden min-w-0 items-center gap-2 text-primary-foreground sm:flex">
              <div className="h-8 w-8 shrink-0 rounded-full bg-cover bg-center" style={{ backgroundImage: `url(${user.bilibili_avatar})` }} />
              <span className="max-w-28 truncate text-sm">{user.bilibili_name}</span>
            </div>
            <button type="button" onClick={logout} className="flex h-9 w-9 items-center justify-center rounded text-primary-foreground hover:bg-white/10" title="退出登录" aria-label="退出登录"><LogOut className="h-4 w-4" /></button>
          </>
        ) : (
          <button type="button" disabled={loading} onClick={() => openAccountDialog('login')} className="flex h-9 shrink-0 items-center gap-1.5 rounded px-2 text-sm text-primary-foreground hover:bg-white/10 disabled:opacity-60 sm:px-3" title="登录 / 注册"><UserRound className="h-4 w-4" /><span className="hidden whitespace-nowrap sm:inline">登录 / 注册</span></button>
        )}
      </div>
    </div>
  );
}
