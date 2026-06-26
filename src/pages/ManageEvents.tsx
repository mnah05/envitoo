import { Link, useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
import { CalendarDays, Globe, MapPin, Plus, Settings2, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Empty,
  EmptyDescription,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { PageContainer } from "@/components/domain/Layout";
import { Skeleton } from "@/components/ui/skeleton";
import { EventCover } from "@/components/domain/EventCard";
import { EventStatusBadge } from "@/components/domain/StatusBadge";
import { useMyEvents, useCancelEvent } from "@/hooks/use-events";
import { formatEventDate } from "@/lib/format";
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

export function ManageEvents() {
  const [includeCancelled, setIncludeCancelled] = useState(false);
  const { data, isLoading } = useMyEvents({ includeCancelled }, true);
  const events = data?.events ?? [];
  const cancelMut = useCancelEvent();
  const navigate = useNavigate();

  useEffect(() => {
    document.title = "My events — Pulse";
  }, []);

  return (
    <div className="pt-16">
      <PageContainer className="py-10">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">My events</h1>
            <p className="mt-2 text-muted-foreground">Events you're hosting.</p>
          </div>
          <Button asChild>
            <Link to="/create">
              <Plus data-icon="inline-start" />
              New event
            </Link>
          </Button>
        </div>

        <div className="mt-6 flex items-center gap-3">
          <Switch
            id="show-cancelled"
            checked={includeCancelled}
            onCheckedChange={setIncludeCancelled}
          />
          <label htmlFor="show-cancelled" className="text-sm text-muted-foreground">
            Show cancelled events
          </label>
        </div>

        <div className="mt-8 flex flex-col gap-4">
          {isLoading ? (
            Array.from({ length: 4 }).map((_, i) => (
              <div
                key={i}
                className="flex flex-col gap-4 rounded-2xl border bg-card p-4 sm:flex-row sm:items-center sm:gap-5"
              >
                <Skeleton className="size-20 shrink-0 rounded-xl sm:size-24" />
                <div className="flex flex-1 flex-col gap-2">
                  <div className="flex items-center gap-2">
                    <Skeleton className="h-5 w-20 rounded-full" />
                    <Skeleton className="h-3 w-10" />
                  </div>
                  <Skeleton className="h-5 w-2/3" />
                  <div className="flex gap-4">
                    <Skeleton className="h-4 w-28" />
                    <Skeleton className="h-4 w-24" />
                  </div>
                </div>
                <div className="flex items-center gap-2 sm:flex-col sm:items-stretch">
                  <Skeleton className="h-8 w-24" />
                  <Skeleton className="h-8 w-24" />
                </div>
              </div>
            ))
          ) : events.length === 0 ? (
            <Empty className="min-h-[40vh] border">
              <EmptyMedia variant="icon">
                <Plus className="size-6" />
              </EmptyMedia>
              <EmptyTitle>No events yet</EmptyTitle>
              <EmptyDescription>Create your first event to start hosting.</EmptyDescription>
              <Button asChild>
                <Link to="/create">Create an event</Link>
              </Button>
            </Empty>
          ) : (
            events.map((event) => (
              <div
                key={event.id}
                className="flex flex-col gap-4 rounded-2xl border bg-card p-4 sm:flex-row sm:items-center sm:gap-5"
              >
                <EventCover
                  event={event}
                  className="size-20 shrink-0 aspect-square sm:size-24"
                  rounded="rounded-xl"
                />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <EventStatusBadge status={event.status} />
                    <span className="text-xs text-muted-foreground">
                      {new Date(event.created_at).getFullYear()}
                    </span>
                  </div>
                  <h2 className="mt-1 truncate text-lg font-semibold">{event.title}</h2>
                  <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
                    <span className="flex items-center gap-1.5">
                      <CalendarDays className="size-4" />
                      {formatEventDate(event.starts_at, event.ends_at)}
                    </span>
                    <span className="flex items-center gap-1.5">
                      {event.is_virtual ? <Globe className="size-4" /> : <MapPin className="size-4" />}
                      {event.is_virtual ? "Virtual" : event.venue_name || "Location"}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-2 sm:flex-col sm:items-stretch">
                  <Button asChild variant="outline" size="sm" className="flex-1">
                    <Link to={`/manage/${event.id}`}>
                      <Settings2 data-icon="inline-start" />
                      Manage
                    </Link>
                  </Button>
                  {event.status !== "cancelled" && (
                    <CancelEventDialog
                      eventTitle={event.title}
                      pending={cancelMut.isPending}
                      onConfirm={() =>
                        cancelMut.mutate(event.id, {
                          onSuccess: () => navigate("/manage"),
                        })
                      }
                    />
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </PageContainer>
    </div>
  );
}

function CancelEventDialog({
  eventTitle,
  pending,
  onConfirm,
}: {
  eventTitle: string;
  pending: boolean;
  onConfirm: () => void;
}) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="sm" className="flex-1 text-muted-foreground">
          Cancel
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-destructive/10 text-destructive">
            <TriangleAlert />
          </AlertDialogMedia>
          <AlertDialogTitle>Cancel this event permanently?</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="flex flex-col gap-4 text-left">
              <p>
                You are about to cancel <span className="font-medium text-foreground">"{eventTitle}"</span>.
              </p>
              <div className="rounded-xl border border-destructive/25 bg-destructive/5 px-4 py-3 text-sm text-foreground">
                <p className="font-semibold text-destructive">This action cannot be undone.</p>
                <p className="mt-1 text-muted-foreground">
                  Once cancelled, the event cannot be republished or restored.
                </p>
              </div>
              <ul className="list-disc space-y-1.5 pl-5 text-muted-foreground">
                <li>Removed from Discover and closed to new registrations</li>
                <li>Existing tickets stay on file — attendees can still view them</li>
                <li>Check-in will be disabled for this event</li>
              </ul>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>Keep event</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={pending}
            onClick={onConfirm}
          >
            {pending ? "Cancelling…" : "Yes, cancel event"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
