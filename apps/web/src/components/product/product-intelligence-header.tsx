import { Button } from "@/components/ui/button";
import { PackageIcon } from "@/components/ui/icons";
import { Card } from "@/components/ui/card";
import {
  fieldValue,
  formatPrice,
  productSubtitle,
  productTitle,
  type Product,
} from "@/lib/product";
import { parseProductImageUrl, productSourceLabel } from "@/lib/product-intelligence";
import { cn } from "@/lib/cn";

type ProductIntelligenceHeaderProps = {
  product: Product;
  syncButtonLabel: string;
  syncDisabled: boolean;
  onSync: () => void;
  lastSyncedLabel: string;
  syncMessage?: string;
  className?: string;
};

export function ProductIntelligenceHeader({
  product,
  syncButtonLabel,
  syncDisabled,
  onSync,
  lastSyncedLabel,
  syncMessage,
  className,
}: ProductIntelligenceHeaderProps) {
  const imageUrl = parseProductImageUrl(product);
  const price = formatPrice(product);
  const source = productSourceLabel(product);
  const sku = fieldValue(product.sku);

  return (
    <Card
      variant="tinted"
      className={cn("overflow-hidden p-0", className)}
      padding="none"
    >
      <div className="flex flex-col gap-6 p-6 sm:flex-row sm:items-start sm:p-8">
        <div className="relative mx-auto h-36 w-36 shrink-0 overflow-hidden rounded-card-lg bg-secondary sm:mx-0 sm:h-40 sm:w-40">
          {imageUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={imageUrl} alt="" className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full w-full flex-col items-center justify-center gap-2 text-muted">
              <PackageIcon size={32} />
              <span className="text-[10px] font-medium uppercase tracking-wider">Görsel yok</span>
            </div>
          )}
        </div>

        <div className="min-w-0 flex-1 space-y-3 text-center sm:text-left">
          <p className="kicker">Product Intelligence</p>
          <h2 className="font-heading text-section-title text-ink">{productTitle(product)}</h2>
          {productSubtitle(product) && (
            <p className="text-sm text-text-secondary">{productSubtitle(product)}</p>
          )}
          <div className="flex flex-wrap items-center justify-center gap-2 sm:justify-start">
            {price && (
              <span className="rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
                {price}
              </span>
            )}
            {source && (
              <span className="rounded-full border border-border bg-surface px-3 py-1 text-xs text-muted">
                Kaynak: {source}
              </span>
            )}
            {sku && (
              <span className="rounded-full border border-border bg-surface px-3 py-1 font-mono text-xs text-muted">
                {sku}
              </span>
            )}
          </div>
        </div>

        <div className="flex shrink-0 flex-col gap-2 sm:items-end">
          <Button variant="secondary" disabled={syncDisabled} onClick={onSync}>
            {syncButtonLabel}
          </Button>
          <p className="max-w-xs text-center text-xs text-muted sm:text-right">{lastSyncedLabel}</p>
          {syncMessage && <p className="max-w-xs text-center text-xs text-text-secondary sm:text-right">{syncMessage}</p>}
        </div>
      </div>
    </Card>
  );
}
