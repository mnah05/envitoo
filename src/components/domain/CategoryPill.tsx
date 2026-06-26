import { cn } from "@/lib/utils";
import { CATEGORY_LABELS } from "@/lib/api";

const CATEGORY_STYLES: Record<string, string> = {
  technical: "bg-sky-50 text-sky-700 ring-sky-200",
  cultural: "bg-violet-50 text-violet-700 ring-violet-200",
  sports: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  workshop: "bg-amber-50 text-amber-700 ring-amber-200",
  seminar: "bg-indigo-50 text-indigo-700 ring-indigo-200",
  hackathon: "bg-rose-50 text-rose-700 ring-rose-200",
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
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset",
        CATEGORY_STYLES[category] ?? CATEGORY_STYLES.other,
        className,
      )}
    >
      {label}
    </span>
  );
}
