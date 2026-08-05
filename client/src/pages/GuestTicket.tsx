import { Link, useSearchParams } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { QRCodeCanvas, QRCodeSVG } from "qrcode.react";
import { CalendarDays, Download, Globe, MapPin, Ticket } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";
import { Skeleton } from "@/components/ui/skeleton";
import { EventStatusBadge, RegStatusBadge } from "@/components/domain/StatusBadge";
import { useEvent } from "@/hooks/use-events";
import { useGuestRegistration, useCancelGuestRegistration } from "@/hooks/use-registrations";
import { formatEventDate, formatFullDate, initials } from "@/lib/format";
import { ApiError } from "@/lib/api";
import { CancelTicketDialog } from "@/components/domain/CancelTicketDialog";
import { generateTicketPdf } from "@/components/domain/TicketPDF";

export function GuestTicket() {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const { data: reg, isLoading, error } = useGuestRegistration(token);
  const { data: event } = useEvent(reg?.event_id ?? "", !!reg?.event_id);
  const cancelMut = useCancelGuestRegistration();
  const qrCanvasRef = useRef<HTMLCanvasElement>(null);
  const [pdfBusy, setPdfBusy] = useState(false);

  async function handleDownload() {
    if (!reg || !event) return;
    const canvas = qrCanvasRef.current;
    if (!canvas) return;
    setPdfBusy(true);
    try {
      const qrDataUrl = canvas.toDataURL("image/png");
      const blob = await generateTicketPdf({ event, reg, token }, qrDataUrl);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `envitoo-ticket-${token.slice(0, 8)}.pdf`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.error("Could not generate PDF");
    } finally {
      setPdfBusy(false);
    }
  }

  useEffect(() => {
    document.title = "Your ticket — EnviToo";
  }, []);

  if (!token) {
    return (
      <div className="pt-14">
        <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
          <Ticket className="size-10 text-muted-foreground" />
          <h1 className="display-title text-2xl">No ticket provided</h1>
          <p className="text-lede text-[1.0625rem]">Check your email for the ticket link.</p>
          <Button asChild variant="outline">
            <Link to="/discover">Discover events</Link>
          </Button>
        </PageContainer>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="pt-20">
        <PageContainer width="narrow" className="py-10">
<div className="mx-auto overflow-hidden rounded-3xl border bg-card shadow-sm">
            <div className="bg-foreground p-6">
              <div className="flex items-center justify-between">
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-5 w-20 rounded-full" />
              </div>
              <Skeleton className="mt-4 h-7 w-2/3" />
              <Skeleton className="mt-2 h-4 w-1/3" />
            </div>
            <div className="grid gap-6 p-6 sm:grid-cols-[1fr_auto] sm:items-center">
              <div className="flex flex-col gap-4">
                <div className="flex items-center gap-3">
                  <Skeleton className="size-11 rounded-full" />
                  <div className="flex flex-col gap-2">
                    <Skeleton className="h-4 w-28" />
                    <Skeleton className="h-3 w-40" />
                  </div>
                </div>
                <div className="flex flex-col gap-2">
                  <Skeleton className="h-4 w-36" />
                  <Skeleton className="h-4 w-28" />
                </div>
              </div>
              <div className="flex flex-col items-center gap-2">
                <Skeleton className="size-40 rounded-2xl" />
                <Skeleton className="h-3 w-24" />
              </div>
            </div>
          <div className="border-t border-border/30" aria-hidden />
            <div className="flex flex-col gap-3 p-6 sm:flex-row sm:items-center sm:justify-between">
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="h-9 w-40" />
            </div>
          </div>
        </PageContainer>
      </div>
    );
  }

  if (error instanceof ApiError && error.status === 404) {
    return (
      <div className="pt-14">
        <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
          <h1 className="display-title text-2xl">Ticket not found</h1>
          <p className="text-lede max-w-sm text-[1.0625rem]">
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
        <div className="mx-auto overflow-hidden rounded-3xl border border-self-strong bg-card">
          {/* Ticket header */}
          <div className="relative bg-foreground p-6 text-background">
            <div className="pointer-events-none absolute -right-8 -top-8 size-32 rounded-full bg-primary/15 blur-3xl" />
            <div className="relative flex items-center justify-between">
              <span className="text-ui inline-flex items-center gap-2 text-[13px]">
                <Ticket className="size-4" /> EnviToo Ticket
              </span>
              <RegStatusBadge status={reg.status} />
            </div>
            <h1 className="display-title relative mt-4 text-2xl">
              {event?.title ?? "Your event"}
            </h1>
            {event && (
              <p className="relative mt-1 text-[0.9375rem] tracking-[-0.01em] text-background/80">
                {formatFullDate(event.starts_at)}
              </p>
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
                <span className="text-ui grid size-11 place-items-center rounded-full bg-primary/10 text-primary">
                  {initials(reg.name)}
                </span>
                <div>
                  <p className="display-headline text-[1.0625rem]">{reg.name}</p>
                  <p className="text-[0.9375rem] tracking-[-0.01em] text-muted-foreground">
                    {reg.email}
                  </p>
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

          <div className="border-t border-border/30" />

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
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  onClick={handleDownload}
                  disabled={pdfBusy || !event}
                >
                  <Download className="size-4" />
                  {pdfBusy ? "Generating…" : "Download ticket"}
                </Button>
                <CancelTicketDialog
                  pending={cancelMut.isPending}
                  onConfirm={() => cancelMut.mutate(token)}
                  triggerLabel="Cancel registration"
                  triggerSize="default"
                />
              </div>
            )}
          </div>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Lost the link? Search your inbox for the registration confirmation.
        </p>

        {/* High-resolution QR for PDF export — visually hidden but rendered. */}
        <div aria-hidden className="sr-only">
          <QRCodeCanvas ref={qrCanvasRef} value={ticketUrl} size={320} level="M" />
        </div>
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
