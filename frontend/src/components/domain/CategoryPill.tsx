import { cn } from "@/lib/utils";
import { CATEGORY_LABELS } from "@/lib/api";

const CATEGORY_STYLES: Record<string, string> = {
  technical: "bg-sky-500/10 text-sky-400 ring-sky-500/20",
  cultural: "bg-rose-500/10 text-rose-400 ring-rose-500/20",
  sports: "bg-emerald-500/10 text-emerald-400 ring-emerald-500/20",
  workshop: "bg-amber-500/10 text-amber-400 ring-amber-500/20",
  seminar: "bg-teal-500/10 text-teal-400 ring-teal-500/20",
  hackathon: "bg-orange-500/10 text-orange-400 ring-orange-500/20",
  other: "bg-muted text-muted-foreground ring-border",
};

export function CategoryPill({
  category,
  className,
}: {
  category: string;
  className?: string;
}) {
  const label = CATEGORY_LABELS[category] ?? category;
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-3 py-1 text-xs font-medium ring-1 ring-inset",
        CATEGORY_STYLES[category] ?? CATEGORY_STYLES.other,
        className,
      )}
    >
      {label}
    </span>
  );
}
