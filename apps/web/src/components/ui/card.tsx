import { HTMLAttributes, forwardRef } from "react";
import { cn } from "@/lib/cn";

export type CardVariant = "default" | "interactive" | "tinted" | "mint";

type CardProps = HTMLAttributes<HTMLDivElement> & {
  variant?: CardVariant;
  padding?: "none" | "sm" | "md" | "lg";
};

const variantClass: Record<CardVariant, string> = {
  default: "card",
  interactive: "card-interactive",
  tinted: "card-tinted",
  mint: "card-mint",
};

const paddingClass = {
  none: "p-0",
  sm: "p-4",
  md: "p-6",
  lg: "p-8",
};

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, variant = "default", padding, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(variantClass[variant], padding && paddingClass[padding], className)}
      {...props}
    >
      {children}
    </div>
  ),
);

Card.displayName = "Card";

export function CardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mb-4 space-y-1", className)} {...props} />;
}

export function CardTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  return <h3 className={cn("card-title", className)} {...props} />;
}

export function CardDescription({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("text-sm text-muted", className)} {...props} />;
}

export function CardContent({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("", className)} {...props} />;
}

export function CardFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mt-4 flex items-center gap-2", className)} {...props} />;
}
