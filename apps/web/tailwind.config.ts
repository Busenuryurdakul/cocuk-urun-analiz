import type { Config } from "tailwindcss";

const withAlpha = (variable: string) => `rgb(var(${variable}) / <alpha-value>)`;

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        ink: withAlpha("--ink"),
        muted: withAlpha("--muted"),
        cream: withAlpha("--cream"),
        paper: withAlpha("--paper"),
        sand: withAlpha("--sand"),
        "on-brand": withAlpha("--on-brand"),
        forest: {
          DEFAULT: withAlpha("--forest"),
          deep: withAlpha("--forest-deep"),
          mid: withAlpha("--forest-mid"),
          soft: withAlpha("--forest-soft"),
        },
        clay: {
          DEFAULT: withAlpha("--clay"),
          deep: withAlpha("--clay-deep"),
          soft: withAlpha("--clay-soft"),
        },
        miyuna: {
          50: "#f0f6f4",
          100: "#dce8e3",
          200: "#b8d0c7",
          300: "#8bb3a6",
          400: "#5a8f7e",
          500: "#2d6a5a",
          600: "#1e4d45",
          700: "#183e38",
          800: "#12302b",
          900: "#0c211e",
        },
      },
      fontFamily: {
        sans: ["var(--font-sans)", "ui-sans-serif", "system-ui", "sans-serif"],
        display: ["var(--font-display)", "ui-serif", "Georgia", "serif"],
      },
      boxShadow: {
        card: "var(--shadow-card)",
        lift: "var(--shadow-lift)",
      },
      borderRadius: {
        "2.5xl": "1.25rem",
      },
      opacity: {
        12: "0.12",
        15: "0.15",
        85: "0.85",
      },
    },
  },
  plugins: [],
};

export default config;
