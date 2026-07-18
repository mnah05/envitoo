import { cn } from "@/lib/utils";

const REG_STATUS_STYLES: Record<string, string> = {
  pending: "text-amber-400",
  approved: "text-emerald-400",
  rejected: "text-rose-400",
  cancelled: "text-muted-foreground",
};

const REG_STATUS_DOTS: Record<string, string> = {
  pending: "bg-amber-400",
  approved: "bg-emerald-400",
  rejected: "bg-rose-400",
  cancelled: "bg-muted-foreground",
};

export function RegStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-sm", className)}>
      <span
        className={cn("size-1.5 rounded-full", REG_STATUS_DOTS[status] ?? "bg-muted-foreground")}
        aria-hidden
      />
      <span className={cn("font-medium capitalize", REG_STATUS_STYLES[status])}>
        {status}
      </span>
    </span>
  );
}

const EVENT_STATUS_STYLES: Record<string, string> = {
  published: "text-emerald-400",
  cancelled: "text-rose-400",
  completed: "text-muted-foreground",
};

const EVENT_STATUS_DOTS: Record<string, string> = {
  published: "bg-emerald-400",
  cancelled: "bg-rose-400",
  completed: "bg-muted-foreground",
};

export function EventStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-sm", className)}>
      <span
        className={cn("size-1.5 rounded-full", EVENT_STATUS_DOTS[status] ?? "bg-muted-foreground")}
        aria-hidden
      />
      <span className={cn("font-medium capitalize", EVENT_STATUS_STYLES[status])}>
        {status}
      </span>
    </span>
  );
}

const VOLUNTEER_APP_STATUS_STYLES: Record<string, string> = {
  pending: "text-amber-400",
  approved: "text-emerald-400",
  rejected: "text-rose-400",
  cancelled: "text-muted-foreground",
};

const VOLUNTEER_APP_STATUS_DOTS: Record<string, string> = {
  pending: "bg-amber-400",
  approved: "bg-emerald-400",
  rejected: "bg-rose-400",
  cancelled: "bg-muted-foreground",
};

export function VolunteerAppStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-sm", className)}>
      <span
        className={cn(
          "size-1.5 rounded-full",
          VOLUNTEER_APP_STATUS_DOTS[status] ?? "bg-muted-foreground",
        )}
        aria-hidden
      />
      <span className={cn("font-medium capitalize", VOLUNTEER_APP_STATUS_STYLES[status])}>
        {status}
      </span>
    </span>
  );
}
