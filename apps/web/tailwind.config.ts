import type { Config } from "tailwindcss";

const withAlpha = (variable: string) => `rgb(var(${variable}) / <alpha-value>)`;

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        base: withAlpha("--bg-base"),
        secondary: withAlpha("--bg-secondary"),
        mint: {
          DEFAULT: withAlpha("--mint"),
          surface: withAlpha("--bg-mint"),
        },
        primary: {
          DEFAULT: withAlpha("--primary"),
          hover: withAlpha("--primary-hover"),
          soft: withAlpha("--purple-soft"),
        },
        ink: withAlpha("--text-primary"),
        muted: withAlpha("--text-muted"),
        "text-secondary": withAlpha("--text-secondary"),
        surface: withAlpha("--surface"),
        border: withAlpha("--border"),
        safe: withAlpha("--safe"),
        caution: withAlpha("--caution"),
        risk: withAlpha("--risk"),
        info: withAlpha("--info"),
        "on-brand": withAlpha("--on-brand"),
        cream: withAlpha("--bg-base"),
        paper: withAlpha("--surface"),
        sand: withAlpha("--border"),
        forest: {
          DEFAULT: withAlpha("--primary"),
          deep: withAlpha("--sidebar-bg"),
          mid: withAlpha("--purple-soft"),
          soft: withAlpha("--bg-secondary"),
        },
        clay: {
          DEFAULT: withAlpha("--caution"),
          deep: withAlpha("--caution"),
          soft: withAlpha("--bg-mint"),
        },
        miyuna: {
          50: "#F7F5FC",
          100: "#EAE7F1",
          200: "#D4CFF0",
          300: "#A89CF6",
          400: "#8B7DF0",
          500: "#7567E8",
          600: "#6657D8",
          700: "#5548C4",
          800: "#4439A8",
          900: "#1E2330",
        },
      },
      fontFamily: {
        sans: ["var(--font-body)", "ui-sans-serif", "system-ui", "sans-serif"],
        heading: ["var(--font-heading)", "ui-sans-serif", "system-ui", "sans-serif"],
        display: ["var(--font-heading)", "ui-sans-serif", "system-ui", "sans-serif"],
      },
      fontSize: {
        display: ["2.75rem", { lineHeight: "1.1", letterSpacing: "-0.02em", fontWeight: "600" }],
        "page-title": ["2rem", { lineHeight: "1.2", letterSpacing: "-0.02em", fontWeight: "600" }],
        "section-title": ["1.5rem", { lineHeight: "1.3", letterSpacing: "-0.01em", fontWeight: "600" }],
        "card-title": ["1.0625rem", { lineHeight: "1.4", fontWeight: "600" }],
        supporting: ["0.8125rem", { lineHeight: "1.5" }],
      },
      maxWidth: {
        page: "80rem",
      },
      boxShadow: {
        card: "var(--shadow-card)",
        "card-hover": "var(--shadow-card-hover)",
        lift: "var(--shadow-lift)",
        focus: "var(--shadow-focus)",
      },
      borderRadius: {
        "2.5xl": "1.25rem",
        card: "1.375rem",
        "card-lg": "1.5rem",
      },
      transitionDuration: {
        fast: "150ms",
        normal: "200ms",
        slow: "250ms",
      },
    },
  },
  plugins: [],
};

export default config;
