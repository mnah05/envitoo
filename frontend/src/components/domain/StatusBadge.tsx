import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const REG_STATUS_STYLES: Record<string, string> = {
  pending: "bg-amber-500/10 text-amber-400 ring-amber-500/20",
  approved: "bg-emerald-500/10 text-emerald-400 ring-emerald-500/20",
  rejected: "bg-rose-500/10 text-rose-400 ring-rose-500/20",
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
  published: "bg-emerald-500/10 text-emerald-400 ring-emerald-500/20",
  cancelled: "bg-rose-500/10 text-rose-400 ring-rose-500/20",
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

const VOLUNTEER_APP_STATUS_STYLES: Record<string, string> = {
  pending: "bg-amber-500/10 text-amber-400 ring-amber-500/20",
  approved: "bg-emerald-500/10 text-emerald-400 ring-emerald-500/20",
  rejected: "bg-rose-500/10 text-rose-400 ring-rose-500/20",
  cancelled: "bg-muted text-muted-foreground ring-border",
};

export function VolunteerAppStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  return (
    <Badge
      variant="outline"
      className={cn("capitalize ring-1", VOLUNTEER_APP_STATUS_STYLES[status], className)}
    >
      {status}
    </Badge>
  );
}
