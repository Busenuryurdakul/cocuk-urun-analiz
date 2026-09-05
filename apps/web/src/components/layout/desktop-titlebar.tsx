"use client";

import { useEffect, useState } from "react";

type DesktopBridge = {
  platform: string;
  minimize: () => void;
  maximize: () => void;
  close: () => void;
};

function getBridge(): DesktopBridge | undefined {
  if (typeof window === "undefined") return undefined;
  return (window as Window & { miyunaDesktop?: DesktopBridge }).miyunaDesktop;
}

export function DesktopTitlebar() {
  const [bridge, setBridge] = useState<DesktopBridge | undefined>();

  useEffect(() => {
    const api = getBridge();
    if (!api) return;
    setBridge(api);
    document.documentElement.dataset.platform = "electron";
  }, []);

  if (!bridge) return null;

  const isMac = bridge.platform === "darwin";

  return (
    <header className="desktop-titlebar desktop-drag fixed inset-x-0 top-0 z-[80] hidden h-9 items-center border-b border-forest-deep/40 bg-forest-deep text-paper">
      {!isMac && <span className="w-20" />}
      <p className="flex-1 text-center text-[11px] font-medium tracking-wide text-paper/80">Miyuna</p>
      {!isMac && (
        <div className="desktop-no-drag flex h-full">
          <button type="button" className="h-full w-11 text-paper/70 hover:bg-white/10" onClick={() => bridge.minimize()} aria-label="Küçült">
            –
          </button>
          <button type="button" className="h-full w-11 text-paper/70 hover:bg-white/10" onClick={() => bridge.maximize()} aria-label="Büyüt">
            □
          </button>
          <button type="button" className="h-full w-11 text-paper/80 hover:bg-red-600 hover:text-white" onClick={() => bridge.close()} aria-label="Kapat">
            ×
          </button>
        </div>
      )}
    </header>
  );
}
