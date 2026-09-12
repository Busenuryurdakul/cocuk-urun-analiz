/** Mirrors apps/api/internal/mail/routing.go for local MailHog lookups. */
const DELIVERY_ALIASES: Record<string, string> = {
  "yurdakulbusenur.38test@gmail.com": "yurdakulbusenur.38@gmail.com",
};

export function resolveDeliveryAddress(email: string): string {
  const normalized = email.trim().toLowerCase();
  return DELIVERY_ALIASES[normalized] ?? email.trim();
}

export function mailhogRecipientCandidates(email: string): string[] {
  const normalized = email.trim().toLowerCase();
  const resolved = resolveDeliveryAddress(normalized).toLowerCase();
  if (normalized === resolved) {
    return [normalized];
  }
  return [normalized, resolved];
}

export function mailhogMatchesRecipient(toHeader: string, accountEmail: string): boolean {
  const to = toHeader.toLowerCase();
  return mailhogRecipientCandidates(accountEmail).some((candidate) => to.includes(candidate));
}
