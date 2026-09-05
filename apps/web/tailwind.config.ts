import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        ink: "#1a1916",
        muted: "#4f4942",
        cream: "#f8f5f0",
        paper: "#fffdf9",
        sand: "#ebe4da",
        forest: {
          DEFAULT: "#1e4d45",
          deep: "#14352f",
          mid: "#2d6a5a",
          soft: "#dce8e3",
        },
        clay: {
          DEFAULT: "#c45c26",
          deep: "#9a4518",
          soft: "#f4e0d2",
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
        card: "0 1px 0 rgba(26,25,22,0.04), 0 18px 40px -24px rgba(20,53,47,0.35)",
        lift: "0 24px 60px -28px rgba(20,53,47,0.45)",
      },
      borderRadius: {
        "2.5xl": "1.25rem",
      },
    },
  },
  plugins: [],
};

export default config;
