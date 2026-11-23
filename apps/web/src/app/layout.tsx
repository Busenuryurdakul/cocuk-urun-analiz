import type { Metadata } from "next";
import { Inter, Plus_Jakarta_Sans } from "next/font/google";
import { DesktopBridge } from "@/components/layout/desktop-bridge";
import { DesktopTitlebar } from "@/components/layout/desktop-titlebar";
import { Providers } from "@/components/providers";
import { THEME_INIT_SCRIPT } from "@/lib/theme";
import "./globals.css";

const body = Inter({
  subsets: ["latin", "latin-ext"],
  variable: "--font-body",
  display: "swap",
});

const heading = Plus_Jakarta_Sans({
  subsets: ["latin", "latin-ext"],
  variable: "--font-heading",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Miyuna — Çocuk ürünleri için analiz",
  description:
    "Çocuk ürünleri için agent destekli analiz ve karar destek platformu. Bir skora değil, kaynağına güven.",
  icons: { icon: "/favicon.svg" },
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="tr" suppressHydrationWarning className={`${body.variable} ${heading.variable}`}>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_INIT_SCRIPT }} />
      </head>
      <body className="font-sans">
        <Providers>
          <DesktopBridge />
          <DesktopTitlebar />
          {children}
        </Providers>
      </body>
    </html>
  );
}
