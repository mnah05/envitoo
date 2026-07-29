import { Link } from "react-router-dom";
import { CalendarDays, ClipboardList, Globe, MapPin } from "lucide-react";
import { useReducedMotion } from "framer-motion";
import { ClipboardCheckIcon } from "lucide-animated";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { RoleBadge } from "@/components/domain/RoleBadge";
import { VolunteerAppStatusBadge } from "@/components/domain/StatusBadge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { useMyVolunteerApplications, useCancelVolunteerApplication } from "@/hooks/use-volunteers";
import { useEvent } from "@/hooks/use-events";
import { formatCardDate, formatEventDate } from "@/lib/format";
import type { VolunteerApplication } from "@/lib/api";

export function MyVolunteerApplications() {
  const { data, isLoading } = useMyVolunteerApplications({ limit: 50 });
  const apps = data?.applications ?? [];
  const reduceMotion = useReducedMotion();

  return (
    <div className="flex flex-col gap-3">
      {isLoading ? (
        Array.from({ length: 3 }).map((_, i) => (
          <div
            key={i}
            className="flex flex-col gap-4 rounded-xl border border-border bg-card p-5 sm:flex-row sm:items-center sm:justify-between"
          >
            <div className="flex flex-1 flex-col gap-2">
              <Skeleton className="h-5 w-2/3" />
              <div className="flex gap-4">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-24" />
              </div>
            </div>
            <div className="flex items-center gap-3 sm:flex-col sm:items-end">
              <Skeleton className="h-5 w-20 rounded-full" />
              <Skeleton className="h-8 w-20" />
            </div>
          </div>
        ))
      ) : apps.length === 0 ? (
        <Empty className="min-h-[40vh] rounded-xl border border-border">
          <EmptyMedia variant="icon">
            <ClipboardCheckIcon size={24} animateOnHover={!reduceMotion} />
          </EmptyMedia>
          <EmptyTitle>No volunteer applications yet</EmptyTitle>
          <EmptyDescription>
            Find an event you&apos;d like to help with and apply to volunteer.
          </EmptyDescription>
          <Button asChild>
            <Link to="/discover">Discover events</Link>
          </Button>
        </Empty>
      ) : (
        apps.map((app) => <ApplicationRow key={app.id} app={app} />)
      )}
    </div>
  );
}

function ApplicationRow({ app }: { app: VolunteerApplication }) {
  const { data: event, isLoading } = useEvent(app.event_id);
  const cancelMut = useCancelVolunteerApplication();
  const canCancel = app.status === "pending";

  return (
    <div className="flex flex-col gap-4 rounded-2xl border bg-card p-5 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        {isLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Skeleton className="h-4 w-32" />
          </div>
        ) : event ? (
          <>
            <div className="flex flex-wrap items-center gap-2">
              <Link
                to={`/e/${event.slug}`}
                className="display-headline truncate text-lg hover:text-primary"
              >
                {event.title}
              </Link>
              {app.is_admin && <RoleBadge isAdmin />}
            </div>
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">{app.role_name}</span>
              <span className="flex items-center gap-1.5">
                <CalendarDays className="size-4" />
                {formatEventDate(event.starts_at, event.ends_at)}
              </span>
              <span className="flex items-center gap-1.5">
                {event.is_virtual ? <Globe className="size-4" /> : <MapPin className="size-4" />}
                {event.is_virtual ? "Virtual" : event.venue_name || "Location"}
              </span>
            </div>
          </>
        ) : (
          <span className="text-sm text-muted-foreground">Event details unavailable</span>
        )}
      </div>

      <div className="flex items-center gap-3 sm:flex-col sm:items-end">
        <VolunteerAppStatusBadge status={app.status} />
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            Applied {formatCardDate(app.created_at)}
          </span>
          {canCancel && (
            <CancelAppDialog
              pending={cancelMut.isPending}
              onConfirm={() => cancelMut.mutate(app.id)}
            />
          )}
        </div>
      </div>
    </div>
  );
}

function CancelAppDialog({
  pending,
  onConfirm,
}: {
  pending: boolean;
  onConfirm: () => void;
}) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="outline" size="sm">
          Cancel
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-destructive/10 text-destructive">
            <ClipboardList />
          </AlertDialogMedia>
          <AlertDialogTitle>Cancel this application?</AlertDialogTitle>
          <AlertDialogDescription>
            You can re-apply later if you change your mind.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>Keep it</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={onConfirm}>
            {pending ? "Cancelling…" : "Yes, cancel"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
