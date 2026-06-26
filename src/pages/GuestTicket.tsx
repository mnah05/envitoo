import { Link, useSearchParams } from "react-router-dom";
import { useEffect } from "react";
import { QRCodeSVG } from "qrcode.react";
import { CalendarDays, Globe, MapPin, Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { PageContainer, Loading } from "@/components/domain/Layout";
import { EventStatusBadge, RegStatusBadge } from "@/components/domain/StatusBadge";
import { useEvent } from "@/hooks/use-events";
import { useGuestRegistration, useCancelGuestRegistration } from "@/hooks/use-registrations";
import { formatEventDate, formatFullDate, initials } from "@/lib/format";
import { ApiError } from "@/lib/api";

export function GuestTicket() {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const { data: reg, isLoading, error } = useGuestRegistration(token);
  const { data: event } = useEvent(reg?.event_id ?? "", !!reg?.event_id);
  const cancelMut = useCancelGuestRegistration();

  useEffect(() => {
    document.title = "Your ticket — Pulse";
  }, []);

  if (!token) {
    return (
      <div className="pt-16">
        <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
          <Ticket className="size-10 text-muted-foreground" />
          <h1 className="text-2xl font-bold">No ticket provided</h1>
          <p className="text-muted-foreground">Check your email for the ticket link.</p>
          <Button asChild variant="outline">
            <Link to="/discover">Discover events</Link>
          </Button>
        </PageContainer>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="pt-16">
        <Loading label="Loading your ticket" />
      </div>
    );
  }

  if (error instanceof ApiError && error.status === 404) {
    return (
      <div className="pt-16">
        <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
          <h1 className="text-2xl font-bold">Ticket not found</h1>
          <p className="text-muted-foreground">
            This ticket link is invalid or has expired.
          </p>
          <Button asChild variant="outline">
            <Link to="/discover">Discover events</Link>
          </Button>
        </PageContainer>
      </div>
    );
  }

  if (!reg) return null;

  const ticketCancelled = reg.status === "cancelled";
  const eventCancelled = event?.status === "cancelled";
  const ticketUrl = `${window.location.origin}/ticket?token=${token}`;

  return (
    <div className="pt-20">
      <PageContainer width="narrow" className="py-10">
        <div className="mx-auto overflow-hidden rounded-3xl border bg-card shadow-sm">
          {/* Ticket header */}
          <div className="relative bg-foreground p-6 text-background">
            <div className="pointer-events-none absolute -right-10 -top-10 size-40 rounded-full bg-primary/30 blur-2xl" />
            <div className="relative flex items-center justify-between">
              <span className="inline-flex items-center gap-2 text-sm font-medium">
                <Ticket className="size-4" /> Pulse Ticket
              </span>
              <RegStatusBadge status={reg.status} />
            </div>
            <h1 className="relative mt-4 text-2xl font-bold tracking-tight">
              {event?.title ?? "Your event"}
            </h1>
            {event && (
              <p className="relative mt-1 text-background/80">{formatFullDate(event.starts_at)}</p>
            )}
          </div>

          {eventCancelled && (
            <div className="border-b bg-amber-50 px-6 py-4 text-sm text-amber-900">
              <div className="flex flex-wrap items-center gap-2">
                <EventStatusBadge status="cancelled" />
                <span>This event has been cancelled by the organizer.</span>
              </div>
            </div>
          )}

          {/* Ticket body */}
          <div className="grid gap-6 p-6 sm:grid-cols-[1fr_auto] sm:items-center">
            <div className="flex flex-col gap-4">
              <div className="flex items-center gap-3">
                <span className="grid size-11 place-items-center rounded-full bg-primary/10 font-semibold text-primary">
                  {initials(reg.name)}
                </span>
                <div>
                  <p className="font-semibold">{reg.name}</p>
                  <p className="text-sm text-muted-foreground">{reg.email}</p>
                </div>
              </div>

              {event && (
                <div className="flex flex-col gap-2 text-sm">
                  <Row icon={<CalendarDays className="size-4" />}>
                    {formatEventDate(event.starts_at, event.ends_at)}
                  </Row>
                  <Row icon={event.is_virtual ? <Globe className="size-4" /> : <MapPin className="size-4" />}>
                    {event.is_virtual ? "Virtual event" : event.venue_name || "Location"}
                  </Row>
                </div>
              )}
            </div>

            {/* QR */}
            <div className="flex flex-col items-center gap-2">
              <div className="rounded-2xl border bg-white p-3">
                <QRCodeSVG value={ticketUrl} size={140} level="M" />
              </div>
              <p className="max-w-[12rem] break-all text-center text-xs text-muted-foreground">
                {token.slice(0, 8)}…{token.slice(-6)}
              </p>
            </div>
          </div>

          <Separator />

          {/* Actions */}
          <div className="flex flex-col gap-3 p-6 sm:flex-row sm:items-center sm:justify-between">
            <div className="text-sm text-muted-foreground">
              {ticketCancelled
                ? "This ticket has been cancelled."
                : eventCancelled
                  ? "Your registration is on file, but the event will not take place."
                  : "Present this QR at check-in. Keep this link safe — it's your only proof of registration."}
            </div>
            {!ticketCancelled && (
              <Button
                variant="outline"
                disabled={cancelMut.isPending}
                onClick={() => cancelMut.mutate(token)}
              >
                {cancelMut.isPending ? "Cancelling…" : "Cancel registration"}
              </Button>
            )}
          </div>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Lost the link? Search your inbox for the registration confirmation.
        </p>
      </PageContainer>
    </div>
  );
}

function Row({ icon, children }: { icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2 text-muted-foreground">
      <span className="text-foreground">{icon}</span>
      <span className="text-foreground">{children}</span>
    </div>
  );
}
