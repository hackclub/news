import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: "media",
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        background: "var(--color-background)",
        foreground: "var(--color-foreground)",
        primary: "var(--color-primary)",
        secondary: "var(--color-secondary)",
        accent: "var(--color-accent)",
        muted: "var(--color-muted)",
        red: "var(--color-red)",
        gray: "var(--color-gray)",
        blue: "var(--color-blue)",
        green: "var(--color-green)",
        cyan: "var(--color-cyan)",
        orange: "var(--color-orange)",
        yellow: "var(--color-yellow)",
        snow: "var(--color-snow)",
        white: "var(--color-white)",
        smoke: "var(--color-smoke)",
        slate: "var(--color-slate)",
        steel: "var(--color-steel)",
        black: "var(--color-black)",
        darkless: "var(--color-darkless)",
        dark: "var(--color-dark)",
        darker: "var(--color-darker)",
        purple: "var(--color-purple)",
      },
      fontFamily: {
        sans: ["var(--font-body)", "ui-sans-serif", "system-ui"],
        serif: ["var(--font-body)", "ui-serif", "Georgia"],
        mono: ["var(--font-body)", "ui-monospace", "SFMono-Regular"],
      },
    },
  },
  plugins: [],
};
export default config;
