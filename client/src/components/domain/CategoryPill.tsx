import { cn } from "@/lib/utils";
import { CATEGORY_LABELS } from "@/lib/api";

const CATEGORY_STYLES: Record<string, string> = {
  technical: "text-sky-500 dark:text-sky-400",
  cultural: "text-zinc-600 dark:text-zinc-300",
  sports: "text-emerald-600 dark:text-emerald-400",
  workshop: "text-amber-600 dark:text-amber-400",
  seminar: "text-blue-600 dark:text-blue-400",
  hackathon: "text-orange-600 dark:text-orange-400",
  other: "text-muted-foreground",
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
        "inline-flex items-center rounded-md bg-card/80 px-2.5 py-0.5 text-xs font-medium backdrop-blur",
        CATEGORY_STYLES[category] ?? CATEGORY_STYLES.other,
        className,
      )}
    >
      {label}
    </span>
  );
}
