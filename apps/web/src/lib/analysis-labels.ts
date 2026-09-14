const POLICY_SENTENCES: Array<[string, string]> = [
  [
    "Risk signals exist but the match or evidence is not strong enough to block automatically.",
    "Risk işaretleri var ancak eşleşme veya kanıt, otomatik engelleme için yeterince güçlü değil.",
  ],
  [
    "Official confirmed recall matched this product. Do not treat it as safe.",
    "Resmi ve doğrulanmış bir geri çağırma bu ürünle eşleşti. Güvenli kabul etmeyin.",
  ],
  [
    "High-severity finding is backed by strong persisted evidence.",
    "Yüksek şiddetli bir bulgu güçlü kanıtla destekleniyor.",
  ],
  [
    "Sources disagree. A human reviewer should inspect the evidence.",
    "Kaynaklar birbiriyle çelişiyor. Bir incelemenin kanıtı kontrol etmesi gerekir.",
  ],
  [
    "Supported medium-severity issues were found. Proceed with the listed warnings.",
    "Desteklenen orta düzey sorunlar bulundu. Listelenen uyarılara dikkat edin.",
  ],
  [
    "Organization policy requires extra review when any safety finding is present.",
    "Kuruluş politikası, herhangi bir güvenlik bulgusu olduğunda ek inceleme istiyor.",
  ],
  [
    "No significant supported safety risk was confirmed.",
    "Desteklenen önemli bir güvenlik riski doğrulanmadı.",
  ],
  ["No approved reviews were available.", "Onaylanmış kullanıcı yorumu bulunamadı."],
  [
    "Unverified recall language was removed from the confirmed recall list.",
    "Doğrulanmamış geri çağırma ifadeleri onaylı listeden çıkarıldı.",
  ],
];

const STATUS_LABELS: Record<string, string> = {
  PENDING: "Bekliyor",
  RUNNING: "Çalışıyor",
  COMPLETED: "Tamamlandı",
  FAILED: "Başarısız",
  REJECTED: "Reddedildi",
  CANCELLED: "İptal edildi",
  ORCHESTRATOR_DISPATCH_FAILED: "Analiz servisine ulaşılamadı",
  MAX_RECOVERY_ATTEMPTS: "Kurtarma denemesi aşıldı",
  STALE_LEASE: "Analiz zaman aşımına uğradı",
};

const RISK_LABELS: Record<string, string> = {
  LOW: "Düşük",
  MEDIUM: "Orta",
  HIGH: "Yüksek",
  CRITICAL: "Kritik",
};

const FINDING_TYPE_LABELS: Record<string, string> = {
  CHOKING: "Boğulma riski",
  SUFFOCATION: "Nefes alma riski",
  STRANGULATION: "Boğma / ip riski",
  CHEMICAL: "Kimyasal risk",
  FIRE: "Yangın riski",
  ELECTRICAL: "Elektrik riski",
  STRUCTURAL: "Yapısal risk",
  AGE_SUITABILITY: "Yaş uygunluğu",
  HYGIENE: "Hijyen",
  INJURY: "Yaralanma riski",
  RECALL: "Geri çağırma",
  MISLEADING_CLAIM: "Yanıltıcı iddia",
  OTHER: "Diğer",
  INSUFFICIENT_EVIDENCE: "Yetersiz kanıt",
};

const FINDING_TYPE_EXPLANATIONS: Record<string, string> = {
  INSUFFICIENT_EVIDENCE:
    "Otomatik karar için yeterli doğrulanmış kaynak yok. Bu bir güvenlik tehlikesi değil; inceleme önerilir.",
};

const SUPPORT_STATUS_LABELS: Record<string, string> = {
  SUPPORTED: "Destekleniyor",
  PARTIALLY_SUPPORTED: "Kısmen destekleniyor",
  UNSUPPORTED: "Desteklenmiyor",
  CONTRADICTED: "Çelişiyor",
};

const INSIGHT_KIND_LABELS: Record<string, string> = {
  POSITIVE: "Olumlu",
  NEGATIVE: "Olumsuz",
  SAFETY: "Güvenlik",
  QUALITY: "Kalite",
  DURABILITY: "Dayanıklılık",
  USABILITY: "Kullanım",
  ASSEMBLY: "Kurulum",
  AGE: "Yaş",
  REPEATED: "Tekrarlanan şikayet",
};

function humanizeCode(value: string) {
  return value.replaceAll("_", " ").toLowerCase();
}

export function statusLabel(status: string) {
  return STATUS_LABELS[status] ?? humanizeCode(status);
}

export function riskLabel(risk: string) {
  return RISK_LABELS[risk] ?? humanizeCode(risk);
}

export function findingTypeLabel(type: string) {
  return FINDING_TYPE_LABELS[type] ?? humanizeCode(type);
}

export function isQualityFinding(type: string) {
  return type === "INSUFFICIENT_EVIDENCE";
}

export function findingRationale(type: string, rationale: string) {
  const trimmed = rationale.trim();
  if (!trimmed || trimmed === type || FINDING_TYPE_LABELS[trimmed]) {
    return FINDING_TYPE_EXPLANATIONS[type] ?? "";
  }
  return localizePolicyText(trimmed);
}

export function supportStatusLabel(status: string) {
  return SUPPORT_STATUS_LABELS[status] ?? humanizeCode(status);
}

export function insightKindLabel(kind: string) {
  return INSIGHT_KIND_LABELS[kind] ?? humanizeCode(kind);
}

export function localizePolicyText(text: string) {
  let out = text.trim();
  for (const [english, turkish] of POLICY_SENTENCES) {
    out = out.replaceAll(english, turkish);
  }
  return out;
}

export function decisionLabel(decision: string) {
  switch (decision) {
    case "ALLOW":
      return "İzin verildi";
    case "ALLOW_WITH_WARNING":
      return "Uyarı ile izin";
    case "REVIEW_REQUIRED":
      return "İnceleme gerekli";
    case "BLOCK":
      return "Engellendi";
    default:
      return decision;
  }
}
