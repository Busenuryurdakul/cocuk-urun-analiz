import { cn } from "@/lib/cn";

type MetricCardProps = {
  label: string;
  value: string | number;
  change?: number;
  changeLabel?: string;
  context?: string;
  trend?: number[];
  className?: string;
};

function MiniTrend({ data }: { data: number[] }) {
  const max = Math.max(...data, 1);
  const min = Math.min(...data, 0);
  const range = max - min || 1;
  const w = 64;
  const h = 24;
  const points = data
    .map((v, i) => {
      const x = (i / (data.length - 1)) * w;
      const y = h - ((v - min) / range) * h;
      return `${x},${y}`;
    })
    .join(" ");

  const trendUp = data.length >= 2 && data[data.length - 1] >= data[0];

  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} className="shrink-0" aria-hidden>
      <polyline
        points={points}
        fill="none"
        stroke={trendUp ? "rgb(var(--safe))" : "rgb(var(--risk))"}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        opacity={0.7}
      />
    </svg>
  );
}

export function MetricCard({
  label,
  value,
  change,
  changeLabel,
  context,
  trend,
  className,
}: MetricCardProps) {
  const changePositive = change !== undefined && change >= 0;

  return (
    <div
      className={cn(
        "rounded-card border border-border/80 bg-surface p-5 shadow-card transition-all duration-normal hover:-translate-y-0.5 hover:shadow-card-hover",
        className,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm font-medium text-text-secondary">{label}</p>
        {trend && trend.length >= 2 && <MiniTrend data={trend} />}
      </div>
      <p className="mt-2 font-heading text-3xl font-semibold tracking-tight text-ink">{value}</p>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        {change !== undefined && (
          <span
            className={cn(
              "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
              changePositive ? "bg-safe/10 text-safe" : "bg-risk/10 text-risk",
            )}
          >
            {changePositive ? "+" : ""}
            {change}%
            {changeLabel && <span className="ml-1 opacity-70">{changeLabel}</span>}
          </span>
        )}
        {context && <span className="text-[11px] text-muted">{context}</span>}
      </div>
    </div>
  );
}
