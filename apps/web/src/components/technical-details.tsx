"use client";

import { useId } from "react";

type TechnicalDetailsProps = {
  label: string;
  value: string;
};

/** Collapsible technical error/details block for analysis surfaces. */
export function TechnicalDetails({ label, value }: TechnicalDetailsProps) {
  const trimmed = value.trim();
  const panelId = useId();
  if (!trimmed) {
    return null;
  }

  return (
    <details className="max-w-full rounded-xl border border-clay/20 bg-white/60 px-3 py-2 text-sm">
      <summary
        className="cursor-pointer list-none font-medium text-muted outline-none marker:content-none focus-visible:rounded-lg focus-visible:ring-2 focus-visible:ring-forest/30 [&::-webkit-details-marker]:hidden"
        aria-controls={panelId}
      >
        {label}
      </summary>
      <pre
        id={panelId}
        className="mt-2 max-w-full overflow-x-auto whitespace-pre-wrap break-words font-mono text-xs text-clay"
      >
        {trimmed}
      </pre>
    </details>
  );
}
