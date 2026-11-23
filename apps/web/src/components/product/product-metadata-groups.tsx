import { fieldValue, type Product, type ProductFieldKey } from "@/lib/product";
import { Card, CardTitle } from "@/components/ui/card";

const GROUPS: { title: string; keys: ProductFieldKey[] }[] = [
  {
    title: "Ürün bilgileri",
    keys: ["description", "materials", "safetyWarnings"],
  },
  {
    title: "Kimlik / kaynak",
    keys: ["sku", "brand", "seller"],
  },
  {
    title: "Kategori / yaş",
    keys: ["category", "targetAge"],
  },
  {
    title: "Teknik alanlar",
    keys: ["currentPrice", "originalPrice", "currency", "rating", "reviewCount", "stockStatus", "attributes"],
  },
];

function FieldRow({ label, value, missing }: { label: string; value: string | null; missing: boolean }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-border/50 py-2 last:border-0 sm:flex-row sm:justify-between">
      <dt className="text-xs font-medium uppercase tracking-wide text-muted">{label}</dt>
      <dd className="text-sm text-ink">
        {value ?? <span className="text-muted">{missing ? "Kaynakta yok" : "—"}</span>}
      </dd>
    </div>
  );
}

const FIELD_LABELS: Record<ProductFieldKey, string> = {
  name: "Ad",
  brand: "Marka",
  category: "Kategori",
  description: "Açıklama",
  targetAge: "Hedef yaş",
  materials: "Malzeme",
  safetyWarnings: "Güvenlik uyarıları",
  currentPrice: "Güncel fiyat",
  originalPrice: "Liste fiyatı",
  currency: "Para birimi",
  seller: "Satıcı",
  rating: "Puan",
  reviewCount: "Yorum sayısı",
  attributes: "Özellikler",
  imageRefs: "Görseller",
  stockStatus: "Stok",
  sku: "SKU",
};

export function ProductMetadataGroups({ product }: { product: Product }) {
  return (
    <Card padding="md">
      <CardTitle className="text-base">Ürün verileri</CardTitle>
      <div className="mt-4 space-y-2">
        {GROUPS.map((group) => {
          const rows = group.keys
            .map((key) => ({
              key,
              label: FIELD_LABELS[key],
              value: fieldValue(product[key]),
              missing: product[key]?.missing ?? false,
            }))
            .filter((row) => row.value || row.missing);

          if (rows.length === 0) return null;

          return (
            <details key={group.title} className="rounded-xl border border-border/60 bg-base/50 px-4 py-2">
              <summary className="cursor-pointer list-none py-2 text-sm font-medium text-ink marker:content-none [&::-webkit-details-marker]:hidden">
                {group.title}
              </summary>
              <dl className="pb-2 pt-1">
                {rows.map((row) => (
                  <FieldRow key={row.key} label={row.label} value={row.value} missing={row.missing} />
                ))}
              </dl>
            </details>
          );
        })}
      </div>
    </Card>
  );
}
