/**
 * Miyuna Design System — Phase 1 tokens
 * Source of truth for programmatic access; CSS vars in globals.css are canonical for styling.
 */

export const colors = {
  bgBase: "#FCFBF8",
  bgSecondary: "#F7F5FC",
  bgMint: "#F3FAF7",
  surface: "#FFFFFF",
  primary: "#7567E8",
  primaryHover: "#6657D8",
  purpleSoft: "#A89CF6",
  mint: "#63CFAE",
  textPrimary: "#1E2330",
  textSecondary: "#626A7A",
  textMuted: "#9096A5",
  border: "#EAE7F1",
  safe: "#46B486",
  caution: "#E7AA3C",
  risk: "#DC686D",
  info: "#5C86E8",
} as const;

export const spacing = {
  1: "4px",
  2: "8px",
  3: "12px",
  4: "16px",
  6: "24px",
  8: "32px",
  10: "40px",
  12: "48px",
} as const;

export const radius = {
  sm: "8px",
  md: "12px",
  lg: "16px",
  xl: "20px",
  card: "22px",
  pill: "9999px",
} as const;

export const typography = {
  display: { size: "44px", lineHeight: "1.1", weight: 600 },
  pageTitle: { size: "32px", lineHeight: "1.2", weight: 600 },
  sectionTitle: { size: "24px", lineHeight: "1.3", weight: 600 },
  cardTitle: { size: "17px", lineHeight: "1.4", weight: 600 },
  body: { size: "15px", lineHeight: "1.6", weight: 400 },
  supporting: { size: "12px", lineHeight: "1.5", weight: 400 },
} as const;

export const motion = {
  fast: "150ms",
  normal: "200ms",
  slow: "250ms",
  easing: "cubic-bezier(0.4, 0, 0.2, 1)",
} as const;

export const layout = {
  sidebarWidth: "240px",
  topbarHeight: "56px",
  pageMaxWidth: "1280px",
  gridColumns: 12,
} as const;

export const shadows = {
  card: "0 1px 2px rgba(30, 35, 48, 0.04), 0 8px 24px -8px rgba(117, 103, 232, 0.08)",
  cardHover: "0 4px 12px rgba(30, 35, 48, 0.06), 0 16px 32px -12px rgba(117, 103, 232, 0.12)",
  focus: "0 0 0 3px rgba(117, 103, 232, 0.18)",
} as const;
