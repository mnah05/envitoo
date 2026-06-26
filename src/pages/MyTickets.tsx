import { Link } from "react-router-dom";
import { useEffect } from "react";
import { CalendarDays, Globe, MapPin, Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import {
  Empty,
  EmptyDescription,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { PageContainer, Loading } from "@/components/domain/Layout";
import { EventStatusBadge, RegStatusBadge } from "@/components/domain/StatusBadge";
import { useMyRegistrations, useCancelRegistration } from "@/hooks/use-registrations";
import { useEvent } from "@/hooks/use-events";
import { formatEventDate } from "@/lib/format";
import type { Registration } from "@/lib/api";

export function MyTickets() {
  const { data, isLoading } = useMyRegistrations({ limit: 50 });
  const regs = data?.registrations ?? [];

  useEffect(() => {
    document.title = "My tickets — Pulse";
  }, []);

  return (
    <div className="pt-16">
      <PageContainer className="py-10">
        <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">My tickets</h1>
        <p className="mt-2 text-muted-foreground">Events you've registered for.</p>

        <div className="mt-8 flex flex-col gap-4">
          {isLoading ? (
            <Loading />
          ) : regs.length === 0 ? (
            <Empty className="min-h-[40vh] border">
              <EmptyMedia variant="icon">
                <Ticket className="size-6" />
              </EmptyMedia>
              <EmptyTitle>No tickets yet</EmptyTitle>
              <EmptyDescription>
                Find an event you love and grab a seat.
              </EmptyDescription>
              <Button asChild>
                <Link to="/discover">Discover events</Link>
              </Button>
            </Empty>
          ) : (
            regs.map((reg) => <TicketRow key={reg.id} reg={reg} />)
          )}
        </div>
      </PageContainer>
    </div>
  );
}

function TicketRow({ reg }: { reg: Registration }) {
  const { data: event, isLoading } = useEvent(reg.event_id);
  const cancelMut = useCancelRegistration();
  const eventCancelled = event?.status === "cancelled";
  const canCancel =
    !eventCancelled && (reg.status === "approved" || reg.status === "pending");

  return (
    <div className="flex flex-col gap-4 rounded-2xl border bg-card p-5 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        {isLoading ? (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Spinner className="size-4" /> Loading event…
          </div>
        ) : event ? (
          <>
            <div className="flex flex-wrap items-center gap-2">
              <Link
                to={`/e/${event.slug}`}
                className="truncate text-lg font-semibold hover:text-primary"
              >
                {event.title}
              </Link>
              {eventCancelled && <EventStatusBadge status="cancelled" />}
            </div>
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
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
        <RegStatusBadge status={reg.status} />
        {canCancel && (
          <Button
            variant="outline"
            size="sm"
            disabled={cancelMut.isPending}
            onClick={() => cancelMut.mutate(reg.id)}
          >
            {cancelMut.isPending ? "Cancelling…" : "Cancel"}
          </Button>
        )}
      </div>
    </div>
  );
}
