import { Link } from "react-router-dom";
import { CalendarDays, Settings2, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { EventCover } from "@/components/domain/EventCard";
import { EventStatusBadge } from "@/components/domain/StatusBadge";
import { RoleBadge } from "@/components/domain/RoleBadge";
import { useMyEventStaff } from "@/hooks/use-volunteers";
import { formatEventDate } from "@/lib/format";

export function MyEventStaffList() {
  const { data, isLoading } = useMyEventStaff({ limit: 50 });
  const events = data?.events ?? [];

  return (
    <div className="flex flex-col gap-4">
      {isLoading ? (
        Array.from({ length: 3 }).map((_, i) => (
          <div
            key={i}
            className="flex flex-col gap-4 rounded-2xl border bg-card p-4 sm:flex-row sm:items-center sm:gap-5"
          >
            <Skeleton className="size-20 shrink-0 rounded-xl sm:size-24" />
            <div className="flex flex-1 flex-col gap-2">
              <Skeleton className="h-5 w-2/3" />
              <div className="flex gap-4">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-28" />
              </div>
            </div>
            <Skeleton className="h-8 w-24" />
          </div>
        ))
      ) : events.length === 0 ? (
        <Empty className="min-h-[40vh] border">
          <EmptyMedia variant="icon">
            <ShieldCheck className="size-6" />
          </EmptyMedia>
          <EmptyTitle>You're not a volunteer admin for any events</EmptyTitle>
          <EmptyDescription>
            Browse events and apply to volunteer. If approved as an admin, you'll help manage them here.
          </EmptyDescription>
          <Button asChild>
            <Link to="/discover">Discover events</Link>
          </Button>
        </Empty>
      ) : (
        events.map((item) => (
          <div
            key={item.event_id}
            className="flex flex-col gap-4 rounded-2xl border bg-card p-4 sm:flex-row sm:items-center sm:gap-5"
          >
            <EventCover
              event={{
                id: item.event_id,
                cover_image_url: null,
                title: item.title,
              } as never}
              className="size-20 shrink-0 aspect-square sm:size-24"
              rounded="rounded-xl"
            />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <EventStatusBadge status={item.event_status} />
                {item.is_admin && <RoleBadge isAdmin />}
              </div>
              <h2 className="mt-1 truncate text-lg font-semibold">{item.title}</h2>
              <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  <CalendarDays className="size-4" />
                  {formatEventDate(item.starts_at, item.ends_at)}
                </span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                Your role: <span className="font-medium text-foreground">{item.role_name}</span>
              </p>
            </div>
            <Button asChild variant="outline" size="sm">
              <Link to={`/manage/${item.event_id}`}>
                <Settings2 data-icon="inline-start" />
                Manage
              </Link>
            </Button>
          </div>
        ))
      )}
    </div>
  );
}
