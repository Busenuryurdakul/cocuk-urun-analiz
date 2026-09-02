import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Miyuna",
  description:
    "Çocuk ürünleri için agent destekli analiz ve karar destek platformu. Skoru değil, skorun kanıtını göster.",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="tr">
      <body>{children}</body>
    </html>
  );
}
