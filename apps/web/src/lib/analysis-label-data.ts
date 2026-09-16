import type { Locale } from "@/lib/i18n/locale-config";

type LabelTable = Partial<Record<Locale, Record<string, string>>> & { en: Record<string, string>; tr: Record<string, string> };

const status = {
  tr: {
    PENDING: "Bekliyor", RUNNING: "Çalışıyor", COMPLETED: "Tamamlandı", FAILED: "Başarısız",
    REJECTED: "Reddedildi", CANCELLED: "İptal edildi", ORCHESTRATOR_DISPATCH_FAILED: "Analiz servisine ulaşılamadı",
    MAX_RECOVERY_ATTEMPTS: "Kurtarma denemesi aşıldı", STALE_LEASE: "Analiz zaman aşımına uğradı",
  },
  en: {
    PENDING: "Pending", RUNNING: "Running", COMPLETED: "Completed", FAILED: "Failed",
    REJECTED: "Rejected", CANCELLED: "Cancelled", ORCHESTRATOR_DISPATCH_FAILED: "Could not reach analysis service",
    MAX_RECOVERY_ATTEMPTS: "Recovery attempts exceeded", STALE_LEASE: "Analysis timed out",
  },
  de: {
    PENDING: "Ausstehend", RUNNING: "Läuft", COMPLETED: "Abgeschlossen", FAILED: "Fehlgeschlagen",
    REJECTED: "Abgelehnt", CANCELLED: "Abgebrochen", ORCHESTRATOR_DISPATCH_FAILED: "Analysedienst nicht erreichbar",
    MAX_RECOVERY_ATTEMPTS: "Wiederherstellungsversuche überschritten", STALE_LEASE: "Analyse-Zeitüberschreitung",
  },
  fr: {
    PENDING: "En attente", RUNNING: "En cours", COMPLETED: "Terminé", FAILED: "Échoué",
    REJECTED: "Rejeté", CANCELLED: "Annulé", ORCHESTRATOR_DISPATCH_FAILED: "Service d'analyse inaccessible",
    MAX_RECOVERY_ATTEMPTS: "Tentatives de reprise dépassées", STALE_LEASE: "Délai d'analyse dépassé",
  },
  es: {
    PENDING: "Pendiente", RUNNING: "En curso", COMPLETED: "Completado", FAILED: "Fallido",
    REJECTED: "Rechazado", CANCELLED: "Cancelado", ORCHESTRATOR_DISPATCH_FAILED: "Servicio de análisis no disponible",
    MAX_RECOVERY_ATTEMPTS: "Intentos de recuperación excedidos", STALE_LEASE: "Tiempo de análisis agotado",
  },
  ar: {
    PENDING: "قيد الانتظار", RUNNING: "قيد التشغيل", COMPLETED: "مكتمل", FAILED: "فشل",
    REJECTED: "مرفوض", CANCELLED: "ملغى", ORCHESTRATOR_DISPATCH_FAILED: "تعذّر الوصول لخدمة التحليل",
    MAX_RECOVERY_ATTEMPTS: "تجاوز محاولات الاست recovery", STALE_LEASE: "انتهت مهلة التحليل",
  },
  zh: {
    PENDING: "等待中", RUNNING: "运行中", COMPLETED: "已完成", FAILED: "失败",
    REJECTED: "已拒绝", CANCELLED: "已取消", ORCHESTRATOR_DISPATCH_FAILED: "无法连接分析服务",
    MAX_RECOVERY_ATTEMPTS: "超过恢复尝试次数", STALE_LEASE: "分析超时",
  },
  ja: {
    PENDING: "保留中", RUNNING: "実行中", COMPLETED: "完了", FAILED: "失敗",
    REJECTED: "拒否", CANCELLED: "キャンセル", ORCHESTRATOR_DISPATCH_FAILED: "分析サービスに到達不可",
    MAX_RECOVERY_ATTEMPTS: "復旧試行超過", STALE_LEASE: "分析タイムアウト",
  },
  ru: {
    PENDING: "Ожидание", RUNNING: "Выполняется", COMPLETED: "Завершено", FAILED: "Ошибка",
    REJECTED: "Отклонено", CANCELLED: "Отменено", ORCHESTRATOR_DISPATCH_FAILED: "Сервис анализа недоступен",
    MAX_RECOVERY_ATTEMPTS: "Превышено число попыток", STALE_LEASE: "Таймаут анализа",
  },
  pt: {
    PENDING: "Pendente", RUNNING: "Em execução", COMPLETED: "Concluído", FAILED: "Falhou",
    REJECTED: "Rejeitado", CANCELLED: "Cancelado", ORCHESTRATOR_DISPATCH_FAILED: "Serviço de análise indisponível",
    MAX_RECOVERY_ATTEMPTS: "Tentativas de recuperação excedidas", STALE_LEASE: "Tempo de análise esgotado",
  },
} satisfies LabelTable;

const risk = {
  tr: { LOW: "Düşük", MEDIUM: "Orta", HIGH: "Yüksek", CRITICAL: "Kritik" },
  en: { LOW: "Low", MEDIUM: "Medium", HIGH: "High", CRITICAL: "Critical" },
  de: { LOW: "Niedrig", MEDIUM: "Mittel", HIGH: "Hoch", CRITICAL: "Kritisch" },
  fr: { LOW: "Faible", MEDIUM: "Moyen", HIGH: "Élevé", CRITICAL: "Critique" },
  es: { LOW: "Bajo", MEDIUM: "Medio", HIGH: "Alto", CRITICAL: "Crítico" },
  ar: { LOW: "منخفض", MEDIUM: "متوسط", HIGH: "مرتفع", CRITICAL: "حرج" },
  zh: { LOW: "低", MEDIUM: "中", HIGH: "高", CRITICAL: "严重" },
  ja: { LOW: "低", MEDIUM: "中", HIGH: "高", CRITICAL: "重大" },
  ru: { LOW: "Низкий", MEDIUM: "Средний", HIGH: "Высокий", CRITICAL: "Критический" },
  pt: { LOW: "Baixo", MEDIUM: "Médio", HIGH: "Alto", CRITICAL: "Crítico" },
} satisfies LabelTable;

const decision = {
  tr: { ALLOW: "İzin verildi", ALLOW_WITH_WARNING: "Uyarı ile izin", REVIEW_REQUIRED: "İnceleme gerekli", BLOCK: "Engellendi" },
  en: { ALLOW: "Allowed", ALLOW_WITH_WARNING: "Allowed with warning", REVIEW_REQUIRED: "Review required", BLOCK: "Blocked" },
  de: { ALLOW: "Erlaubt", ALLOW_WITH_WARNING: "Mit Warnung erlaubt", REVIEW_REQUIRED: "Prüfung erforderlich", BLOCK: "Blockiert" },
  fr: { ALLOW: "Autorisé", ALLOW_WITH_WARNING: "Autorisé avec avertissement", REVIEW_REQUIRED: "Revue requise", BLOCK: "Bloqué" },
  es: { ALLOW: "Permitido", ALLOW_WITH_WARNING: "Permitido con advertencia", REVIEW_REQUIRED: "Revisión requerida", BLOCK: "Bloqueado" },
  ar: { ALLOW: "مسموح", ALLOW_WITH_WARNING: "مسموح مع تحذير", REVIEW_REQUIRED: "مراجعة مطلوبة", BLOCK: "محظور" },
  zh: { ALLOW: "允许", ALLOW_WITH_WARNING: "警告后允许", REVIEW_REQUIRED: "需要审核", BLOCK: "已阻止" },
  ja: { ALLOW: "許可", ALLOW_WITH_WARNING: "警告付き許可", REVIEW_REQUIRED: "レビュー必要", BLOCK: "ブロック" },
  ru: { ALLOW: "Разрешено", ALLOW_WITH_WARNING: "С предупреждением", REVIEW_REQUIRED: "Требуется проверка", BLOCK: "Заблокировано" },
  pt: { ALLOW: "Permitido", ALLOW_WITH_WARNING: "Permitido com aviso", REVIEW_REQUIRED: "Revisão necessária", BLOCK: "Bloqueado" },
} satisfies LabelTable;

export const STATUS_LABELS = status;
export const RISK_LABELS = risk;
export const DECISION_LABELS = decision;

export const FINDING_TYPE_LABELS: LabelTable = {
  tr: {
    CHOKING: "Boğulma riski", SUFFOCATION: "Nefes alma riski", STRANGULATION: "Boğma / ip riski",
    CHEMICAL: "Kimyasal risk", FIRE: "Yangın riski", ELECTRICAL: "Elektrik riski", STRUCTURAL: "Yapısal risk",
    AGE_SUITABILITY: "Yaş uygunluğu", HYGIENE: "Hijyen", INJURY: "Yaralanma riski", RECALL: "Geri çağırma",
    MISLEADING_CLAIM: "Yanıltıcı iddia", OTHER: "Diğer", INSUFFICIENT_EVIDENCE: "Yetersiz kanıt",
  },
  en: {
    CHOKING: "Choking risk", SUFFOCATION: "Suffocation risk", STRANGULATION: "Strangulation / cord risk",
    CHEMICAL: "Chemical risk", FIRE: "Fire risk", ELECTRICAL: "Electrical risk", STRUCTURAL: "Structural risk",
    AGE_SUITABILITY: "Age suitability", HYGIENE: "Hygiene", INJURY: "Injury risk", RECALL: "Recall",
    MISLEADING_CLAIM: "Misleading claim", OTHER: "Other", INSUFFICIENT_EVIDENCE: "Insufficient evidence",
  },
  de: {
    CHOKING: "Erstickungsgefahr", SUFFOCATION: "Erstickungsrisiko", STRANGULATION: "Strangulationsrisiko",
    CHEMICAL: "Chemisches Risiko", FIRE: "Brandrisiko", ELECTRICAL: "Elektrisches Risiko", STRUCTURAL: "Strukturrisiko",
    AGE_SUITABILITY: "Altersgeeignetheit", HYGIENE: "Hygiene", INJURY: "Verletzungsrisiko", RECALL: "Rückruf",
    MISLEADING_CLAIM: "Irreführende Behauptung", OTHER: "Sonstiges", INSUFFICIENT_EVIDENCE: "Unzureichende Beweise",
  },
  fr: {
    CHOKING: "Risque d'étouffement", SUFFOCATION: "Risque de suffocation", STRANGULATION: "Risque de strangulation",
    CHEMICAL: "Risque chimique", FIRE: "Risque d'incendie", ELECTRICAL: "Risque électrique", STRUCTURAL: "Risque structurel",
    AGE_SUITABILITY: "Adaptation à l'âge", HYGIENE: "Hygiène", INJURY: "Risque de blessure", RECALL: "Rappel",
    MISLEADING_CLAIM: "Allégation trompeuse", OTHER: "Autre", INSUFFICIENT_EVIDENCE: "Preuves insuffisantes",
  },
  es: {
    CHOKING: "Riesgo de asfixia", SUFFOCATION: "Riesgo de sufocación", STRANGULATION: "Riesgo de estrangulación",
    CHEMICAL: "Riesgo químico", FIRE: "Riesgo de incendio", ELECTRICAL: "Riesgo eléctrico", STRUCTURAL: "Riesgo estructural",
    AGE_SUITABILITY: "Adecuación por edad", HYGIENE: "Higiene", INJURY: "Riesgo de lesión", RECALL: "Retiro",
    MISLEADING_CLAIM: "Afirmación engañosa", OTHER: "Otro", INSUFFICIENT_EVIDENCE: "Evidencia insuficiente",
  },
  ar: {
    CHOKING: "خطر الاختناق", SUFFOCATION: "خطر الت suffocation", STRANGULATION: "خطر الخنق",
    CHEMICAL: "خطر كيميائي", FIRE: "خطر حريق", ELECTRICAL: "خطر كهربائي", STRUCTURAL: "خطر هيكلي",
    AGE_SUITABILITY: "ملاءمة العمر", HYGIENE: "النظافة", INJURY: "خطر إصابة", RECALL: "استدعاء",
    MISLEADING_CLAIM: "ادعاء مضلل", OTHER: "أخرى", INSUFFICIENT_EVIDENCE: "أدلة غير كافية",
  },
  zh: {
    CHOKING: "窒息风险", SUFFOCATION: "窒息风险", STRANGULATION: "勒颈/绳带风险",
    CHEMICAL: "化学风险", FIRE: "火灾风险", ELECTRICAL: "电气风险", STRUCTURAL: "结构风险",
    AGE_SUITABILITY: "年龄适用性", HYGIENE: "卫生", INJURY: "伤害风险", RECALL: "召回",
    MISLEADING_CLAIM: "误导性声明", OTHER: "其他", INSUFFICIENT_EVIDENCE: "证据不足",
  },
  ja: {
    CHOKING: "窒息リスク", SUFFOCATION: " suffocation リスク", STRANGULATION: "首絞め/コードリスク",
    CHEMICAL: "化学リスク", FIRE: "火災リスク", ELECTRICAL: "電気リスク", STRUCTURAL: "構造リスク",
    AGE_SUITABILITY: "年齢適合性", HYGIENE: "衛生", INJURY: "負傷リスク", RECALL: "リコール",
    MISLEADING_CLAIM: "誤解を招く表示", OTHER: "その他", INSUFFICIENT_EVIDENCE: "証拠不十分",
  },
  ru: {
    CHOKING: "Риск удушья", SUFFOCATION: "Риск удушения", STRANGULATION: "Риск strangulation",
    CHEMICAL: "Химический риск", FIRE: "Пожарный риск", ELECTRICAL: "Электрический риск", STRUCTURAL: "Структурный риск",
    AGE_SUITABILITY: "Возрастная пригодность", HYGIENE: "Гигиена", INJURY: "Риск травмы", RECALL: "Отзыв",
    MISLEADING_CLAIM: "Вводящее в заблуждение заявление", OTHER: "Другое", INSUFFICIENT_EVIDENCE: "Недостаточно доказательств",
  },
  pt: {
    CHOKING: "Risco de asfixia", SUFFOCATION: "Risco de sufocação", STRANGULATION: "Risco de estrangulamento",
    CHEMICAL: "Risco químico", FIRE: "Risco de incêndio", ELECTRICAL: "Risco elétrico", STRUCTURAL: "Risco estrutural",
    AGE_SUITABILITY: "Adequação etária", HYGIENE: "Higiene", INJURY: "Risco de lesão", RECALL: "Recall",
    MISLEADING_CLAIM: "Alegação enganosa", OTHER: "Outro", INSUFFICIENT_EVIDENCE: "Evidência insuficiente",
  },
};

export const FINDING_TYPE_EXPLANATIONS: Partial<Record<Locale, Record<string, string>>> & { en: Record<string, string>; tr: Record<string, string> } = {
  tr: {
    INSUFFICIENT_EVIDENCE: "Otomatik karar için yeterli doğrulanmış kaynak yok. Bu bir güvenlik tehlikesi değil; inceleme önerilir.",
  },
  en: {
    INSUFFICIENT_EVIDENCE: "Not enough verified sources for an automatic decision. This is not a safety hazard; review is recommended.",
  },
  de: { INSUFFICIENT_EVIDENCE: "Nicht genug verifizierte Quellen für eine automatische Entscheidung. Keine Sicherheitsgefahr; Prüfung empfohlen." },
  fr: { INSUFFICIENT_EVIDENCE: "Pas assez de sources vérifiées pour une décision automatique. Pas un danger; revue recommandée." },
  es: { INSUFFICIENT_EVIDENCE: "No hay fuentes verificadas suficientes. No es un peligro de seguridad; se recomienda revisión." },
  ar: { INSUFFICIENT_EVIDENCE: "لا توجد مصادر موثقة كافية. ليس خطراً أمنياً؛ يُنصح بالمراجعة." },
  zh: { INSUFFICIENT_EVIDENCE: "没有足够的已验证来源。非安全隐患；建议审核。" },
  ja: { INSUFFICIENT_EVIDENCE: "自動判定に十分な検証済みソースがありません。安全上の危険ではなく、レビューを推奨します。" },
  ru: { INSUFFICIENT_EVIDENCE: "Недостаточно проверенных источников. Это не опасность; рекомендуется проверка." },
  pt: { INSUFFICIENT_EVIDENCE: "Fontes verificadas insuficientes. Não é perigo de segurança; revisão recomendada." },
};

export const SUPPORT_STATUS_LABELS: LabelTable = {
  tr: { SUPPORTED: "Destekleniyor", PARTIALLY_SUPPORTED: "Kısmen destekleniyor", UNSUPPORTED: "Desteklenmiyor", CONTRADICTED: "Çelişiyor" },
  en: { SUPPORTED: "Supported", PARTIALLY_SUPPORTED: "Partially supported", UNSUPPORTED: "Unsupported", CONTRADICTED: "Contradicted" },
  de: { SUPPORTED: "Unterstützt", PARTIALLY_SUPPORTED: "Teilweise unterstützt", UNSUPPORTED: "Nicht unterstützt", CONTRADICTED: "Widersprüchlich" },
  fr: { SUPPORTED: "Pris en charge", PARTIALLY_SUPPORTED: "Partiellement pris en charge", UNSUPPORTED: "Non pris en charge", CONTRADICTED: "Contradictoire" },
  es: { SUPPORTED: "Respaldado", PARTIALLY_SUPPORTED: "Parcialmente respaldado", UNSUPPORTED: "No respaldado", CONTRADICTED: "Contradicho" },
  ar: { SUPPORTED: "مدعوم", PARTIALLY_SUPPORTED: "مدعوم جزئياً", UNSUPPORTED: "غير مدعوم", CONTRADICTED: "متناقض" },
  zh: { SUPPORTED: "有支持", PARTIALLY_SUPPORTED: "部分支持", UNSUPPORTED: "不支持", CONTRADICTED: "矛盾" },
  ja: { SUPPORTED: "支持あり", PARTIALLY_SUPPORTED: "部分的に支持", UNSUPPORTED: "不支持", CONTRADICTED: "矛盾" },
  ru: { SUPPORTED: "Поддерживается", PARTIALLY_SUPPORTED: "Частично поддерживается", UNSUPPORTED: "Не поддерживается", CONTRADICTED: "Противоречит" },
  pt: { SUPPORTED: "Suportado", PARTIALLY_SUPPORTED: "Parcialmente suportado", UNSUPPORTED: "Não suportado", CONTRADICTED: "Contraditório" },
};

export const INSIGHT_TOPIC_LABELS: LabelTable = {
  tr: {
    positive: "Olumlu", negative: "Olumsuz", safety: "Güvenlik", quality: "Kalite",
    durability: "Dayanıklılık", usability: "Kullanım", ease_of_use: "Kolay kullanım",
    assembly: "Kurulum", age_mismatch: "Yaş uyumsuzluğu", choking: "Boğulma riski", recall: "Geri çağırma",
  },
  en: {
    positive: "Positive", negative: "Negative", safety: "Safety", quality: "Quality",
    durability: "Durability", usability: "Usability", ease_of_use: "Ease of use",
    assembly: "Assembly", age_mismatch: "Age mismatch", choking: "Choking", recall: "Recall",
  },
};

export const INSIGHT_KIND_LABELS: LabelTable = {
  tr: { POSITIVE: "Olumlu", NEGATIVE: "Olumsuz", SAFETY: "Güvenlik", QUALITY: "Kalite", DURABILITY: "Dayanıklılık", USABILITY: "Kullanım", ASSEMBLY: "Kurulum", AGE: "Yaş", REPEATED: "Tekrarlanan şikayet" },
  en: { POSITIVE: "Positive", NEGATIVE: "Negative", SAFETY: "Safety", QUALITY: "Quality", DURABILITY: "Durability", USABILITY: "Usability", ASSEMBLY: "Assembly", AGE: "Age", REPEATED: "Repeated complaint" },
  de: { POSITIVE: "Positiv", NEGATIVE: "Negativ", SAFETY: "Sicherheit", QUALITY: "Qualität", DURABILITY: "Haltbarkeit", USABILITY: "Bedienbarkeit", ASSEMBLY: "Montage", AGE: "Alter", REPEATED: "Wiederholte Beschwerde" },
  fr: { POSITIVE: "Positif", NEGATIVE: "Négatif", SAFETY: "Sécurité", QUALITY: "Qualité", DURABILITY: "Durabilité", USABILITY: "Utilisabilité", ASSEMBLY: "Montage", AGE: "Âge", REPEATED: "Plainte répétée" },
  es: { POSITIVE: "Positivo", NEGATIVE: "Negativo", SAFETY: "Seguridad", QUALITY: "Calidad", DURABILITY: "Durabilidad", USABILITY: "Usabilidad", ASSEMBLY: "Montaje", AGE: "Edad", REPEATED: "Queja repetida" },
  ar: { POSITIVE: "إيجابي", NEGATIVE: "سلبي", SAFETY: "سلامة", QUALITY: "جودة", DURABILITY: "متانة", USABILITY: "قابلية الاستخدام", ASSEMBLY: "تجميع", AGE: "العمر", REPEATED: "شكوى متكررة" },
  zh: { POSITIVE: "正面", NEGATIVE: "负面", SAFETY: "安全", QUALITY: "质量", DURABILITY: "耐用性", USABILITY: "可用性", ASSEMBLY: "组装", AGE: "年龄", REPEATED: "重复投诉" },
  ja: { POSITIVE: "ポジティブ", NEGATIVE: "ネガティブ", SAFETY: "安全", QUALITY: "品質", DURABILITY: "耐久性", USABILITY: "使いやすさ", ASSEMBLY: "組立", AGE: "年齢", REPEATED: "繰り返しの苦情" },
  ru: { POSITIVE: "Положительный", NEGATIVE: "Отрицательный", SAFETY: "Безопасность", QUALITY: "Качество", DURABILITY: "Долговечность", USABILITY: "Удобство", ASSEMBLY: "Сборка", AGE: "Возраст", REPEATED: "Повторяющаяся жалоба" },
  pt: { POSITIVE: "Positivo", NEGATIVE: "Negativo", SAFETY: "Segurança", QUALITY: "Qualidade", DURABILITY: "Durabilidade", USABILITY: "Usabilidade", ASSEMBLY: "Montagem", AGE: "Idade", REPEATED: "Reclamação repetida" },
};
