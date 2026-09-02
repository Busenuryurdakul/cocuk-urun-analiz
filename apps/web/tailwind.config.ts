import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        miyuna: {
          50: "#f0f9ff",
          600: "#0284c7",
          900: "#0c4a6e",
        },
      },
    },
  },
  plugins: [],
};

export default config;
