import { cn } from "@/lib/utils";

/** Centered loading state for page-level suspense/fetches. */
export function Loading({ label = "Loading" }: { label?: string }) {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4 text-muted-foreground">
      <div className="relative flex items-center justify-center">
        <div className="size-8 animate-pulse rounded-full border-2 border-primary/30 border-t-primary" />
      </div>
      <span className="text-sm tracking-[-0.01em]">{label}…</span>
    </div>
  );
}

/** Standard content width container with consistent horizontal padding. */
export function PageContainer({
  children,
  className,
  width = "default",
}: {
  children: React.ReactNode;
  className?: string;
  width?: "default" | "wide" | "narrow";
}) {
  const max = width === "wide" ? "max-w-7xl" : width === "narrow" ? "max-w-2xl" : "max-w-6xl";
  return (
    <div className={cn("mx-auto w-full px-4 sm:px-6", max, className)}>{children}</div>
  );
}
