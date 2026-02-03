import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Navbar } from "@/components/navbar";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "JoiAsk 提问箱",
  description: "JoiAsk 提问箱",
  icons: {
    icon: "/favicon.png",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" style={{ scrollbarGutter: 'stable' }}>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased fabric-linen`}
      >
        <Navbar />
        {children}
      </body>
    </html>
  );
}
