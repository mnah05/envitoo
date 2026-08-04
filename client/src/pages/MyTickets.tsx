import { Link } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { CalendarDays, ClipboardList, Download, Globe, MapPin, ShieldCheck, Ticket } from "lucide-react";
import { useReducedMotion } from "framer-motion";
import { CalendarDaysIcon } from "lucide-animated";
import { QRCodeCanvas, QRCodeSVG } from "qrcode.react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Empty,
  EmptyDescription,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { PageContainer } from "@/components/domain/Layout";
import { Skeleton } from "@/components/ui/skeleton";
import { EventStatusBadge, RegStatusBadge } from "@/components/domain/StatusBadge";
import { OnboardingChecklist } from "@/components/domain/OnboardingChecklist";
import { MyVolunteerApplications } from "@/components/domain/MyVolunteerApplications";
import { MyEventStaffList } from "@/components/domain/MyEventStaffList";
import { CancelTicketDialog } from "@/components/domain/CancelTicketDialog";
import { generateTicketPdf } from "@/components/domain/TicketPDF";
import { useMyRegistrations, useCancelRegistration } from "@/hooks/use-registrations";
import { useEvent } from "@/hooks/use-events";
import { formatEventDate } from "@/lib/format";
import type { Registration } from "@/lib/api";

export function MyTickets() {
  const { data, isLoading } = useMyRegistrations({ limit: 50 });
  const regs = data?.registrations ?? [];
  const reduceMotion = useReducedMotion();

  useEffect(() => {
    document.title = "My tickets — EnviToo";
  }, []);

  return (
    <div className="pt-14">
      <PageContainer className="py-10">
        <h1 className="display-title text-3xl sm:text-4xl">My dashboard</h1>
        <p className="text-lede mt-3">
          Your tickets, volunteer applications, and events you help manage.
        </p>

        <div className="mt-8">
          <div className="mb-6">
            <OnboardingChecklist />
          </div>
          <Tabs defaultValue="tickets">
            <TabsList>
              <TabsTrigger value="tickets">
                <Ticket data-icon="inline-start" />
                My tickets
              </TabsTrigger>
              <TabsTrigger value="applications">
                <ClipboardList data-icon="inline-start" />
                Applications
              </TabsTrigger>
              <TabsTrigger value="staff">
                <ShieldCheck data-icon="inline-start" />
                Events I help manage
              </TabsTrigger>
            </TabsList>

            <TabsContent value="tickets">
              <div className="mt-6 flex flex-col gap-6">
                {isLoading ? (
                  Array.from({ length: 2 }).map((_, i) => (
                    <Skeleton key={i} className="h-64 w-full rounded-xl" />
                  ))
                ) : regs.length === 0 ? (
                  <Empty className="min-h-[40vh] rounded-xl border border-border">
                    <EmptyMedia variant="icon">
                      <CalendarDaysIcon size={24} animateOnHover={!reduceMotion} />
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
                  regs.map((reg) => <TicketCard key={reg.id} reg={reg} />)
                )}
              </div>
            </TabsContent>

            <TabsContent value="applications">
              <div className="mt-6">
                <MyVolunteerApplications />
              </div>
            </TabsContent>

            <TabsContent value="staff">
              <div className="mt-6">
                <MyEventStaffList />
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </PageContainer>
    </div>
  );
}

function TicketCard({ reg }: { reg: Registration }) {
  const { data: event, isLoading } = useEvent(reg.event_id);
  const cancelMut = useCancelRegistration();
  const eventCancelled = event?.status === "cancelled";
  const canCancel =
    !eventCancelled && (reg.status === "approved" || reg.status === "pending");

  const ticketUrl = reg.access_token
    ? `${window.location.origin}/ticket?token=${reg.access_token}`
    : null;
  const qrCanvasRef = useRef<HTMLCanvasElement>(null);
  const [pdfBusy, setPdfBusy] = useState(false);

  async function handleDownload() {
    if (!event || !reg.access_token) return;
    const canvas = qrCanvasRef.current;
    if (!canvas) return;
    setPdfBusy(true);
    try {
      const qrDataUrl = canvas.toDataURL("image/png");
      const blob = await generateTicketPdf({ event, reg, token: reg.access_token }, qrDataUrl);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `envitoo-ticket-${reg.access_token.slice(0, 8)}.pdf`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      toast.error("Could not generate PDF");
    } finally {
      setPdfBusy(false);
    }
  }

  if (isLoading) {
    return <Skeleton className="h-64 w-full rounded-2xl" />;
  }

  if (!event) {
    return null;
  }

  return (
    <div className="relative overflow-hidden rounded-xl border border-border bg-card">
      {/* Top: hero area */}
      <div className="relative bg-primary/[0.04]">
        <div className="flex flex-col gap-3 p-5 pb-8">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <h3 className="display-headline-sans text-xl font-semibold">
                {event.title}
              </h3>
              <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
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
            <RegStatusBadge status={reg.status} />
          </div>
        </div>
      </div>

      {/* Perforation line with circle cutouts */}
      <div className="relative flex items-center px-5">
        <div className="absolute left-0 top-1/2 size-5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-background" />
        <div className="h-px flex-1 border-t border-dashed border-border" />
        <div className="absolute right-0 top-1/2 size-5 translate-x-1/2 -translate-y-1/2 rounded-full bg-background" />
      </div>

      {/* Bottom: QR + details */}
      <div className="flex items-center gap-5 p-5 pt-6">
        <div className="flex-shrink-0 rounded-xl bg-white p-2">
          {ticketUrl ? (
            <QRCodeSVG value={ticketUrl} size={100} level="M" />
          ) : (
            <div className="flex size-[100px] items-center justify-center rounded-lg bg-muted text-xs text-muted-foreground">
              No QR
            </div>
          )}
        </div>
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <div className="text-xs text-muted-foreground">
            Ticket #{reg.id.slice(0, 8)}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {eventCancelled && <EventStatusBadge status="cancelled" />}
          </div>
          <div className="mt-1 flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleDownload}
              disabled={pdfBusy || !ticketUrl}
            >
              <Download className="size-4" />
              {pdfBusy ? "Generating…" : "Download"}
            </Button>
            {canCancel && (
              <CancelTicketDialog
                pending={cancelMut.isPending}
                onConfirm={() => cancelMut.mutate(reg.id)}
              />
            )}
            <Button variant="ghost" size="sm" asChild>
              <Link to={`/e/${event.slug}`}>
                View event
              </Link>
            </Button>
          </div>
        </div>
      </div>

      {/* High-resolution QR for PDF export — visually hidden but rendered. */}
      {ticketUrl && (
        <div aria-hidden className="sr-only">
          <QRCodeCanvas ref={qrCanvasRef} value={ticketUrl} size={320} level="M" />
        </div>
      )}
    </div>
  );
}
