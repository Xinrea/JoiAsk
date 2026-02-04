"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  MessageSquare,
  Tag,
  Users,
  Home,
  LogOut,
} from "lucide-react";
import { useAuth } from "@/lib/auth";

const navItems = [
  { href: "/dashboard", label: "概览", icon: LayoutDashboard },
  { href: "/dashboard/questions", label: "提问管理", icon: MessageSquare },
  { href: "/dashboard/tags", label: "话题管理", icon: Tag },
  { href: "/dashboard/users", label: "账号管理", icon: Users },
];

export function Sidebar() {
  const pathname = usePathname();
  const { logout } = useAuth();

  return (
    <aside
      className="flex h-full w-56 shrink-0 flex-col bg-sidebar text-sidebar-foreground sm:w-60"
      style={{
        boxShadow:
          "inset -1px 0 0 rgba(255,255,255,0.08), 4px 0 24px rgba(0,0,0,0.06), inset 0 0 60px rgba(0,0,0,0.03)",
        transition: "box-shadow 200ms ease",
        backgroundImage: `
          repeating-linear-gradient(
            0deg,
            transparent,
            transparent 2px,
            rgba(255, 255, 255, 0.02) 2px,
            rgba(255, 255, 255, 0.02) 3px
          ),
          repeating-linear-gradient(
            90deg,
            transparent,
            transparent 2px,
            rgba(255, 255, 255, 0.02) 2px,
            rgba(255, 255, 255, 0.02) 3px
          )
        `,
        backgroundSize: "6px 6px",
      }}
    >
      <div
        className="flex h-14 shrink-0 items-center justify-center border-b-[1.5px] border-sidebar-border px-4"
        style={{
          boxShadow:
            "inset 0 1px 0 var(--fabric-stitch), 0 1px 0 rgba(0,0,0,0.1)",
        }}
      >
        <h1 className="text-base font-semibold tracking-tight text-sidebar-foreground">
          JoiAsk 管理后台
        </h1>
      </div>
      <nav className="flex-1 space-y-0.5 overflow-y-auto p-3">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-all duration-[200ms] ease-out",
                isActive
                  ? "bg-sidebar-accent text-sidebar-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.08),inset_0_-1px_0_rgba(0,0,0,0.1)]"
                  : "text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-foreground hover:shadow-[inset_0_1px_0_rgba(255,255,255,0.06)]"
              )}
            >
              <Icon className="h-5 w-5 shrink-0 opacity-90" />
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div
        className="border-t-[1.5px] border-sidebar-border p-3 space-y-0.5"
        style={{
          boxShadow:
            "inset 0 1px 0 rgba(255,255,255,0.06), 0 -1px 0 rgba(0,0,0,0.1)",
        }}
      >
        <a
          href="/"
          target="_blank"
          className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-sidebar-foreground/80 transition-all duration-[200ms] ease-out hover:bg-sidebar-accent hover:text-sidebar-foreground hover:shadow-[inset_0_1px_0_rgba(255,255,255,0.06)]"
        >
          <Home className="h-5 w-5 shrink-0 opacity-90" />
          回到提问箱
        </a>
        <button
          type="button"
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-sidebar-foreground/80 transition-all duration-[200ms] ease-out hover:bg-sidebar-accent hover:text-sidebar-foreground hover:shadow-[inset_0_1px_0_rgba(255,255,255,0.06)]"
        >
          <LogOut className="h-5 w-5 shrink-0 opacity-90" />
          注销
        </button>
      </div>
    </aside>
  );
}
