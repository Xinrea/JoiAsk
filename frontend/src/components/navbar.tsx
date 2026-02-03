'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

const NAV_ROUTES = [
  { label: '提问', path: '/' },
  { label: '话题', path: '/tags' },
  { label: '彩虹屁', path: '/rainbow' },
  { label: '搜索', path: '/search' },
];

export function Navbar() {
  const pathname = usePathname();

  const isActive = (path: string) => {
    if (path === '/') return pathname === '/';
    return pathname.startsWith(path);
  };

  return (
    <div className="navbar fabric-nav h-14 px-2.5 flex items-center justify-between z-[999]">
      <div className="h-full flex items-center">
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
            className={`px-3 py-1.5 mx-1 rounded text-primary-foreground transition-all duration-200 ${
              isActive(path)
                ? 'bg-white/20 border border-dashed border-white/30'
                : 'hover:bg-white/10'
            }`}
          >
            {label}
          </Link>
        ))}
      </div>
    </div>
  );
}
