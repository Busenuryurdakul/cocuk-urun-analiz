import {
  PRODUCT_FIELD_LABELS,
  fieldValue,
  formatPrice,
  missingFieldCount,
  productSubtitle,
  productTitle,
  type Product,
  type ProductFieldKey,
} from "@/lib/product";

export function ProductMetaGrid({ product }: { product: Product }) {
  return (
    <dl className="grid gap-3 sm:grid-cols-2">
      {PRODUCT_FIELD_LABELS.filter(({ key }) => key !== "name" && key !== "description").map(({ key, label }) => (
        <ProductFieldRow key={key} fieldKey={key} label={label} product={product} />
      ))}
    </dl>
  );
}

function ProductFieldRow({
  fieldKey,
  label,
  product,
}: {
  fieldKey: ProductFieldKey;
  label: string;
  product: Product;
}) {
  const field = product[fieldKey];
  const value = fieldValue(field);

  return (
    <div className="rounded-2xl bg-cream px-4 py-3">
      <dt className="flex items-center justify-between gap-2 text-[11px] font-semibold uppercase tracking-wide text-muted">
        <span>{label}</span>
        {field.missing ? <span className="badge-clay">Eksik</span> : null}
      </dt>
      <dd className="mt-1 text-sm text-ink">
        {value ?? <span className="text-muted">Kaynakta yok</span>}
      </dd>
      {field.missing && field.missingReason ? (
        <p className="mt-1 text-xs text-muted">{field.missingReason}</p>
      ) : null}
    </div>
  );
}

export function ProductSummaryStats({ product }: { product: Product }) {
  return (
    <dl className="grid grid-cols-3 gap-3 text-center">
      <Stat value={fieldValue(product.rating) ? `★ ${fieldValue(product.rating)}` : "—"} label="Puan" />
      <Stat value={fieldValue(product.reviewCount) ?? "—"} label="Satış kanalı yorumu" />
      <Stat value={String(missingFieldCount(product))} label="Eksik alan" />
    </dl>
  );
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div className="rounded-2xl bg-cream px-2 py-3">
      <dt className="font-display text-2xl text-forest">{value}</dt>
      <dd className="text-[11px] uppercase tracking-wide text-muted">{label}</dd>
    </div>
  );
}

export function ProductCardMeta({ product }: { product: Product }) {
  const price = formatPrice(product);
  const subtitle = productSubtitle(product);

  return (
    <>
      <p className="font-display text-xl">{productTitle(product)}</p>
      <p className="mt-1 text-sm text-muted">{subtitle || "Kaynak alanları eksik olabilir"}</p>
      <div className="mt-3 flex flex-wrap gap-2">
        {price ? <span className="badge-forest">{price}</span> : <span className="badge-clay">Fiyat eksik</span>}
        {fieldValue(product.rating) ? <span className="badge-muted">★ {fieldValue(product.rating)}</span> : null}
        {product.brand.missing ? <span className="badge-clay">Marka eksik</span> : null}
        {missingFieldCount(product) > 0 ? (
          <span className="badge-muted">{missingFieldCount(product)} eksik alan</span>
        ) : (
          <span className="badge-forest">Alanlar dolu</span>
        )}
      </div>
    </>
  );
}
