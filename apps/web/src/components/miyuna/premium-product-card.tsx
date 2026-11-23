import Link from "next/link";
import { cn } from "@/lib/cn";
import { StatusRibbon, StatusRibbonVariant } from "@/components/miyuna/status-ribbon";
import { Button } from "@/components/ui/button";
import { PackageIcon } from "@/components/ui/icons";

type PremiumProductCardProps = {
  productName: string;
  brand?: string;
  category?: string;
  imageUrl?: string;
  riskLevel?: StatusRibbonVariant;
  confidence?: number;
  recallState?: StatusRibbonVariant;
  lastAnalysis?: string;
  href?: string;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
};

export function PremiumProductCard({
  productName,
  brand,
  category,
  imageUrl,
  riskLevel = "pending",
  confidence,
  recallState,
  lastAnalysis,
  href,
  actionLabel = "View analysis",
  onAction,
  className,
}: PremiumProductCardProps) {
  const content = (
    <div
      className={cn(
        "group relative overflow-hidden rounded-card-lg border border-border/80 bg-surface p-4 shadow-card transition-all duration-normal hover:-translate-y-1 hover:border-primary/20 hover:shadow-card-hover sm:p-5",
        href && "cursor-pointer",
        className,
      )}
    >
      <div className="flex gap-4">
        <div className="relative h-16 w-16 shrink-0 overflow-hidden rounded-xl bg-secondary sm:h-20 sm:w-20">
          {imageUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={imageUrl} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-muted">
              <PackageIcon size={24} />
            </div>
          )}
        </div>

        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-start gap-2">
            {brand && <span className="text-[11px] font-medium uppercase tracking-wider text-muted">{brand}</span>}
            {category && (
              <span className="rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-text-secondary">
                {category}
              </span>
            )}
          </div>
          <h4 className="card-title truncate">{productName}</h4>
          <div className="flex flex-wrap items-center gap-2">
            <StatusRibbon variant={riskLevel} size="sm" />
            {recallState && recallState !== "safe" && (
              <StatusRibbon variant={recallState} size="sm" label="Recall" />
            )}
            {confidence !== undefined && (
              <span className="text-[11px] text-muted">{Math.round(confidence)}% conf.</span>
            )}
          </div>
          {lastAnalysis && (
            <p className="text-[11px] text-muted">Last analysis: {lastAnalysis}</p>
          )}
        </div>
      </div>

      {(href || onAction) && (
        <div className="mt-4 border-t border-border/60 pt-3">
          {href ? (
            <span className="text-sm font-medium text-primary transition group-hover:text-primary-hover">
              {actionLabel} →
            </span>
          ) : (
            <Button variant="ghost" size="sm" onClick={onAction}>
              {actionLabel}
            </Button>
          )}
        </div>
      )}
    </div>
  );

  if (href) {
    return (
      <Link href={href} className="block outline-none focus-visible:rounded-card-lg focus-visible:shadow-focus">
        {content}
      </Link>
    );
  }

  return content;
}
