import { Link, useParams } from "react-router-dom";
import { useEffect } from "react";
import { ArrowLeft, CalendarDays, CheckCircle2, Globe, MapPin, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Empty,
  EmptyDescription,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { PageContainer } from "@/components/domain/Layout";
import { Skeleton } from "@/components/ui/skeleton";
import { EventStatusBadge, RegStatusBadge } from "@/components/domain/StatusBadge";
import { useMyEvents } from "@/hooks/use-events";
import {
  useEventRegistrations,
  useUpdateRegistrationStatus,
  useCheckIn,
} from "@/hooks/use-registrations";
import { formatEventDate } from "@/lib/format";
import {
  regStatus,
  UpdateRegistrationStatusInput,
  type Registration,
  type UpdateRegistrationStatusValues,
} from "@/lib/api";

export function EventDashboard() {
  const { id = "" } = useParams();
  const { data: myEvents, isLoading: eventLoading } = useMyEvents(
    { includeCancelled: true },
    !!id,
  );
  const event = myEvents?.events.find((e) => e.id === id);

  const { data, isLoading: regsLoading } = useEventRegistrations(id, { limit: 100 }, !!id);
  const regs = data?.registrations ?? [];

  const statusMut = useUpdateRegistrationStatus();
  const checkInMut = useCheckIn();

  useEffect(() => {
    document.title = event ? `${event.title} — Manage — Pulse` : "Manage — Pulse";
  }, [event]);

  if (eventLoading && !event) {
    return <EventDashboardSkeleton />;
  }

  if (!event) {
    return (
      <div className="pt-16">
        <PageContainer className="flex min-h-[50vh] flex-col items-center justify-center gap-4 text-center">
          <h1 className="text-2xl font-bold">Event not found</h1>
          <p className="text-muted-foreground">You may not have access to this event.</p>
          <Button asChild variant="outline">
            <Link to="/manage">Back to my events</Link>
          </Button>
        </PageContainer>
      </div>
    );
  }

  const approved = regs.filter((r) => r.status === "approved").length;
  const pending = regs.filter((r) => r.status === "pending").length;
  const checkedIn = regs.filter((r) => r.checked_in_at).length;
  const seatsLeft = Math.max(0, event.capacity - approved);

  return (
    <div className="pt-16">
      <PageContainer width="wide" className="py-10">
        <Button asChild variant="ghost" size="sm" className="mb-4 -ml-2">
          <Link to="/manage">
            <ArrowLeft data-icon="inline-start" />
            My events
          </Link>
        </Button>

        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-3xl font-bold tracking-tight">{event.title}</h1>
              <EventStatusBadge status={event.status} />
            </div>
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
          <Button asChild variant="outline">
            <Link to={`/e/${event.slug}`} target="_blank">
              View public page
            </Link>
          </Button>
        </div>

        {/* Stats */}
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Stat icon={<Users className="size-5" />} label="Capacity" value={event.capacity} />
          <Stat icon={<CheckCircle2 className="size-5" />} label="Approved" value={approved} />
          <Stat icon={<Users className="size-5" />} label="Pending" value={pending} />
          <Stat icon={<CheckCircle2 className="size-5" />} label="Checked in" value={checkedIn} />
        </div>
        {event.status !== "cancelled" && (
          <p className="mt-3 text-sm text-muted-foreground">{seatsLeft} seats remaining.</p>
        )}

        {/* Registrations */}
        <div className="mt-10">
          <h2 className="text-xl font-semibold">Registrations</h2>
          <p className="mt-1 text-sm text-muted-foreground">{regs.length} total</p>

          {regsLoading ? (
            <div className="mt-6 overflow-hidden rounded-2xl border">
              <div className="flex items-center gap-4 border-b bg-card px-4 py-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-3 w-32" />
                <Skeleton className="h-3 w-16" />
                <Skeleton className="h-3 w-20" />
              </div>
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex items-center gap-4 border-b px-4 py-3 last:border-0">
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-5 w-20 rounded-full" />
                  <Skeleton className="h-4 w-16" />
                </div>
              ))}
            </div>
          ) : regs.length === 0 ? (
            <Empty className="mt-6 min-h-[30vh] border">
              <EmptyMedia variant="icon">
                <Users className="size-6" />
              </EmptyMedia>
              <EmptyTitle>No registrations yet</EmptyTitle>
              <EmptyDescription>
                Share your event page to start collecting signups.
              </EmptyDescription>
            </Empty>
          ) : (
            <div className="mt-6 overflow-x-auto rounded-2xl border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Attendee</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Check-in</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {regs.map((reg) => (
                    <RegistrationRow
                      key={reg.id}
                      reg={reg}
                      eventCancelled={event.status === "cancelled"}
                      statusPending={statusMut.isPending}
                      checkInPending={checkInMut.isPending}
                      onStatus={(status) => statusMut.mutate({ id: reg.id, values: { status } })}
                      onCheckIn={() =>
                        reg.access_token &&
                        checkInMut.mutate({ id: reg.id, values: { access_token: reg.access_token } })
                      }
                    />
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      </PageContainer>
    </div>
  );
}

function Stat({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
}) {
  return (
    <div className="rounded-2xl border bg-card p-5">
      <div className="flex items-center gap-2 text-muted-foreground">
        {icon}
        <span className="text-sm font-medium">{label}</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{value}</p>
    </div>
  );
}

function RegistrationRow({
  reg,
  eventCancelled,
  statusPending,
  checkInPending,
  onStatus,
  onCheckIn,
}: {
  reg: Registration;
  eventCancelled: boolean;
  statusPending: boolean;
  checkInPending: boolean;
  onStatus: (status: UpdateRegistrationStatusValues["status"]) => void;
  onCheckIn: () => void;
}) {
  const busy = statusPending || checkInPending;

  function handleStatusChange(next: string) {
    const parsed = UpdateRegistrationStatusInput.safeParse({ status: next });
    if (!parsed.success || parsed.data.status === reg.status) return;
    onStatus(parsed.data.status);
  }

  return (
    <TableRow>
      <TableCell className="font-medium">{reg.name}</TableCell>
      <TableCell className="text-muted-foreground">{reg.email}</TableCell>
      <TableCell>
        <RegStatusBadge status={reg.status} />
      </TableCell>
      <TableCell>
        {reg.checked_in_at ? (
          <span className="inline-flex items-center gap-1.5 text-sm text-emerald-700">
            <CheckCircle2 className="size-4" /> Checked in
          </span>
        ) : (
          <span className="text-sm text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-2">
          <Select value={reg.status} onValueChange={handleStatusChange} disabled={busy}>
            <SelectTrigger size="sm" className="w-36 capitalize">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {regStatus.options.map((status) => (
                <SelectItem key={status} value={status} className="capitalize">
                  {status}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {reg.status === "approved" && !reg.checked_in_at && !eventCancelled && (
            <Button size="xs" onClick={onCheckIn} disabled={busy}>
              {checkInPending ? <Spinner /> : "Check in"}
            </Button>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}

function EventDashboardSkeleton() {
  return (
    <div className="pt-16">
      <PageContainer width="wide" className="py-10">
        <Skeleton className="mb-4 h-8 w-24" />
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <Skeleton className="h-8 w-48" />
              <Skeleton className="h-5 w-20 rounded-full" />
            </div>
            <div className="flex gap-4">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-4 w-24" />
            </div>
          </div>
          <Skeleton className="h-9 w-36" />
        </div>
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="rounded-2xl border bg-card p-5">
              <div className="flex items-center gap-2">
                <Skeleton className="size-5 rounded" />
                <Skeleton className="h-4 w-20" />
              </div>
              <Skeleton className="mt-2 h-7 w-16" />
            </div>
          ))}
        </div>
        <div className="mt-10">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="mt-1 h-4 w-20" />
        </div>
      </PageContainer>
    </div>
  );
}
