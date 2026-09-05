export type ProductField = {
  value?: string | null;
  missing: boolean;
  missingReason?: string | null;
  source?: string | null;
  sourceRecordId?: string | null;
  confidence?: number | null;
  extractedAt?: string | null;
};

export type Product = {
  id: string;
  organizationId: string;
  name: ProductField;
  brand: ProductField;
  category: ProductField;
  description: ProductField;
  targetAge: ProductField;
  materials: ProductField;
  safetyWarnings: ProductField;
  currentPrice: ProductField;
  originalPrice: ProductField;
  currency: ProductField;
  seller: ProductField;
  rating: ProductField;
  reviewCount: ProductField;
  attributes: ProductField;
  imageRefs: ProductField;
  stockStatus: ProductField;
  sku: ProductField;
  createdAt: string;
  updatedAt: string;
};

export const PRODUCT_FIELD_SELECTION = `
  id
  organizationId
  name { value missing missingReason source }
  brand { value missing missingReason source }
  category { value missing missingReason source }
  description { value missing missingReason source }
  targetAge { value missing missingReason source }
  materials { value missing missingReason source }
  safetyWarnings { value missing missingReason source }
  currentPrice { value missing missingReason source }
  originalPrice { value missing missingReason source }
  currency { value missing missingReason source }
  seller { value missing missingReason source }
  rating { value missing missingReason source }
  reviewCount { value missing missingReason source }
  attributes { value missing missingReason source }
  imageRefs { value missing missingReason source }
  stockStatus { value missing missingReason source }
  sku { value missing missingReason source }
  createdAt
  updatedAt
`;

export const PRODUCT_FIELD_LABELS = [
  { key: "name", label: "Ad" },
  { key: "brand", label: "Marka" },
  { key: "category", label: "Kategori" },
  { key: "description", label: "Açıklama" },
  { key: "targetAge", label: "Hedef yaş" },
  { key: "materials", label: "Malzeme" },
  { key: "safetyWarnings", label: "Güvenlik uyarıları" },
  { key: "currentPrice", label: "Güncel fiyat" },
  { key: "originalPrice", label: "Liste fiyatı" },
  { key: "currency", label: "Para birimi" },
  { key: "seller", label: "Satıcı" },
  { key: "rating", label: "Puan" },
  { key: "reviewCount", label: "Yorum sayısı" },
  { key: "attributes", label: "Özellikler" },
  { key: "imageRefs", label: "Görseller" },
  { key: "stockStatus", label: "Stok" },
  { key: "sku", label: "SKU" },
] as const;

export type ProductFieldKey = (typeof PRODUCT_FIELD_LABELS)[number]["key"];

export function fieldValue(field?: ProductField | null): string | null {
  const value = field?.value?.trim();
  return value ? value : null;
}

export function productField(product: Product, key: ProductFieldKey): ProductField {
  return product[key];
}

export function missingFieldCount(product: Product): number {
  return PRODUCT_FIELD_LABELS.filter(({ key }) => product[key]?.missing).length;
}

export function formatPrice(product: Product): string | null {
  const price = fieldValue(product.currentPrice);
  if (!price) {
    return null;
  }
  const currency = fieldValue(product.currency);
  return currency ? `${price} ${currency}` : price;
}

export function productTitle(product: Product): string {
  return (
    fieldValue(product.name) ??
    fieldValue(product.sku) ??
    product.name.sourceRecordId ??
    "İsimsiz ürün"
  );
}

export function productSubtitle(product: Product): string {
  return [fieldValue(product.brand), fieldValue(product.category), fieldValue(product.targetAge)]
    .filter(Boolean)
    .join(" · ");
}
