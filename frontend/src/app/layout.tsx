import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Navbar } from "@/components/navbar";
import { AccountAuthProvider } from "@/lib/account-auth";
import { getSiteConfig } from "@/lib/site-config";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const dynamic = "force-dynamic";

export async function generateMetadata(): Promise<Metadata> {
  const config = await getSiteConfig();
  return {
    title: config.site_name,
    description: config.site_description,
    icons: {
      icon: config.favicon_url,
    },
  };
}

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const config = await getSiteConfig();

  return (
    <html lang="zh-CN" style={{ scrollbarGutter: 'stable' }}>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased fabric-linen`}
      >
        <AccountAuthProvider>
          <Navbar logoUrl={config.logo_url} />
          {children}
          <style id="custom-site-css">{config.custom_css}</style>
        </AccountAuthProvider>
      </body>
    </html>
  );
}
