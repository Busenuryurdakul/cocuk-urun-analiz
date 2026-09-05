import type { Metadata } from "next";
import { Fraunces, Plus_Jakarta_Sans } from "next/font/google";
import { DesktopTitlebar } from "@/components/layout/desktop-titlebar";
import "./globals.css";

const sans = Plus_Jakarta_Sans({
  subsets: ["latin", "latin-ext"],
  variable: "--font-sans",
  display: "swap",
});

const display = Fraunces({
  subsets: ["latin", "latin-ext"],
  variable: "--font-display",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Miyuna — Çocuk ürünleri için kanıtlı analiz",
  description:
    "Çocuk ürünleri için agent destekli analiz ve karar destek platformu. Bir skora değil, kanıtına güven.",
  icons: { icon: "/favicon.svg" },
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="tr" className={`${sans.variable} ${display.variable}`}>
      <body className="font-sans">
        <DesktopTitlebar />
        {children}
      </body>
    </html>
  );
}
