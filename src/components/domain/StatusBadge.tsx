import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const REG_STATUS_STYLES: Record<string, string> = {
  pending: "bg-amber-50 text-amber-700 ring-amber-200",
  approved: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  rejected: "bg-rose-50 text-rose-700 ring-rose-200",
  cancelled: "bg-muted text-muted-foreground ring-border",
};

export function RegStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  return (
    <Badge
      variant="outline"
      className={cn("capitalize ring-1", REG_STATUS_STYLES[status], className)}
    >
      {status}
    </Badge>
  );
}

const EVENT_STATUS_STYLES: Record<string, string> = {
  published: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  cancelled: "bg-rose-50 text-rose-700 ring-rose-200",
  completed: "bg-muted text-muted-foreground ring-border",
};

export function EventStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  return (
    <Badge
      variant="outline"
      className={cn("capitalize ring-1", EVENT_STATUS_STYLES[status], className)}
    >
      {status}
    </Badge>
  );
}
