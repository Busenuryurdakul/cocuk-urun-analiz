import { cn } from "@/lib/cn";

type BreakdownSegment = {
  label: string;
  value: number;
  color: string;
};

type IntelligenceScoreRingProps = {
  score: number;
  maxScore?: number;
  label?: string;
  confidence?: number;
  severity?: "safe" | "caution" | "risk";
  breakdown?: BreakdownSegment[];
  size?: "sm" | "md" | "lg";
  className?: string;
};

const sizeConfig = {
  sm: { outer: 88, stroke: 6, fontSize: "text-xl", labelSize: "text-[10px]" },
  md: { outer: 120, stroke: 7, fontSize: "text-3xl", labelSize: "text-[11px]" },
  lg: { outer: 160, stroke: 8, fontSize: "text-4xl", labelSize: "text-xs" },
};

const severityColors = {
  safe: "#46B486",
  caution: "#E7AA3C",
  risk: "#DC686D",
};

export function IntelligenceScoreRing({
  score,
  maxScore = 100,
  label = "Intelligence",
  confidence,
  severity = "safe",
  breakdown,
  size = "md",
  className,
}: IntelligenceScoreRingProps) {
  const cfg = sizeConfig[size];
  const radius = (cfg.outer - cfg.stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const normalized = Math.min(Math.max(score / maxScore, 0), 1);
  const dashOffset = circumference * (1 - normalized);
  const center = cfg.outer / 2;

  const defaultBreakdown: BreakdownSegment[] = breakdown ?? [
    { label: "Safety", value: normalized * 0.4, color: severityColors.safe },
    { label: "Compliance", value: normalized * 0.35, color: "#7567E8" },
    { label: "Recall", value: normalized * 0.25, color: severityColors[severity] },
  ];

  return (
    <div className={cn("flex flex-col items-center gap-3", className)}>
      <div className="relative" style={{ width: cfg.outer, height: cfg.outer }}>
        <svg width={cfg.outer} height={cfg.outer} className="-rotate-90" aria-hidden>
          <circle
            cx={center}
            cy={center}
            r={radius}
            fill="none"
            stroke="rgb(var(--border))"
            strokeWidth={cfg.stroke}
            opacity={0.5}
          />
          {defaultBreakdown.map((seg, i) => {
            const segLen = circumference * seg.value;
            const offset = defaultBreakdown
              .slice(0, i)
              .reduce((acc, s) => acc + circumference * s.value, 0);
            return (
              <circle
                key={seg.label}
                cx={center}
                cy={center}
                r={radius}
                fill="none"
                stroke={seg.color}
                strokeWidth={cfg.stroke}
                strokeDasharray={`${segLen} ${circumference - segLen}`}
                strokeDashoffset={-offset}
                strokeLinecap="round"
                opacity={0.85}
              />
            );
          })}
          <circle
            cx={center}
            cy={center}
            r={radius - cfg.stroke - 4}
            fill="none"
            stroke={severityColors[severity]}
            strokeWidth={2}
            strokeDasharray={`${circumference * normalized * 0.15} ${circumference}`}
            opacity={0.4}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className={cn("font-heading font-semibold tracking-tight text-ink", cfg.fontSize)}>
            {Math.round(score)}
          </span>
          <span className={cn("text-muted", cfg.labelSize)}>{label}</span>
        </div>
      </div>

      <div className="flex flex-wrap justify-center gap-3">
        {defaultBreakdown.map((seg) => (
          <div key={seg.label} className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: seg.color }} aria-hidden />
            <span className="text-[11px] text-muted">{seg.label}</span>
          </div>
        ))}
        {confidence !== undefined && (
          <div className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-primary/40" aria-hidden />
            <span className="text-[11px] text-muted">{Math.round(confidence)}% conf.</span>
          </div>
        )}
      </div>

      <span className="sr-only">
        {label} score {score} out of {maxScore}
        {confidence !== undefined ? `, confidence ${confidence} percent` : ""}
      </span>
    </div>
  );
}
