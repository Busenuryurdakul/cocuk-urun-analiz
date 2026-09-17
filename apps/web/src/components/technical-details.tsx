"use client";

type TechnicalDetailsProps = {
  label: string;
  value: string;
};

export function TechnicalDetails({ label, value }: TechnicalDetailsProps) {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  return (
    <details className="rounded-xl border border-clay/20 bg-white/60 px-3 py-2 text-sm">
      <summary className="cursor-pointer font-medium text-muted">{label}</summary>
      <pre className="mt-2 overflow-x-auto whitespace-pre-wrap break-words font-mono text-xs text-clay">
        {trimmed}
      </pre>
    </details>
  );
}
