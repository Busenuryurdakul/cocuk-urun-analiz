import Link from "next/link";

type LogoProps = {
  href?: string;
  tone?: "forest" | "paper";
  size?: "sm" | "md";
  wordmark?: boolean;
};

export function Logo({ href = "/", tone = "forest", size = "md", wordmark = true }: LogoProps) {
  const mark = size === "sm" ? "h-8 w-8" : "h-10 w-10";
  const text = size === "sm" ? "text-lg" : "text-xl";
  const color = tone === "paper" ? "text-paper" : "text-forest";

  const inner = (
    <span className={`inline-flex items-center gap-2.5 ${color}`}>
      <span className={`${mark} relative inline-flex items-center justify-center`} aria-hidden>
        <svg viewBox="0 0 40 40" className="h-full w-full" fill="none">
          <rect width="40" height="40" rx="12" className={tone === "paper" ? "fill-paper/10" : "fill-forest"} />
          <path
            d="M10 26V16.2c0-1.4.8-2.2 2-2.2.6 0 1.2.3 1.7.9L20 22.2l6.3-7.3c.5-.6 1.1-.9 1.7-.9 1.2 0 2 .8 2 2.2V26"
            className={tone === "paper" ? "stroke-paper" : "stroke-paper"}
            strokeWidth="2.2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          <circle cx="20" cy="28.4" r="1.6" className={tone === "paper" ? "fill-[#e8a07a]" : "fill-[#c45c26]"} />
        </svg>
      </span>
      {wordmark && (
        <span className={`font-display ${text} font-medium leading-none tracking-tight`}>Miyuna</span>
      )}
    </span>
  );

  if (!href) return inner;
  return (
    <Link href={href} className="inline-flex shrink-0 outline-none focus-visible:ring-2 focus-visible:ring-forest/30">
      {inner}
    </Link>
  );
}
