import { Link, useParams } from "react-router-dom";
import { useCallback } from "react";
import { useEffect, useState } from "react";
import { ArrowLeft, CalendarDays, Check, CheckCircle2, Copy, Globe, HeartHandshake, MapPin, Pencil, ScanLine, Users, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
import { VolunteerDashboard } from "@/components/domain/VolunteerDashboard";
import { CheckInScanner } from "@/components/domain/CheckInScanner";
import { useMyEvents, useEvent } from "@/hooks/use-events";
import { useMyEventStaff } from "@/hooks/use-volunteers";
import {
  useEventRegistrations,
  useUpdateRegistrationStatus,
  useApproveRegistration,
  useRejectRegistration,
  useCheckIn,
  useCheckInByEventToken,
} from "@/hooks/use-registrations";
import { formatEventDate } from "@/lib/format";
import {
  regStatus,
  UpdateRegistrationStatusInput,
  type Registration,
  type UpdateRegistrationStatusValues,
} from "@/lib/api";

const REG_PAGE_SIZE = 100;

export function EventDashboard() {
  const { id = "" } = useParams();
  const { data: myEvents, isLoading: myEventsLoading } = useMyEvents(
    { includeCancelled: true },
    !!id,
  );
  const myEvent = myEvents?.events.find((e) => e.id === id);
  const isOrganizer = !!myEvent;

  const { data: publicEvent, isLoading: publicLoading } = useEvent(
    id,
    !!id && !myEvent && !myEventsLoading,
  );
  const event = myEvent ?? publicEvent;

  const { data: staffData } = useMyEventStaff({}, !!id && !isOrganizer);
  const isVolunteerAdmin =
    !isOrganizer && staffData?.events.some((e) => e.event_id === id) === true;

  const [regPage, setRegPage] = useState(0);
  useEffect(() => {
    setRegPage(0);
  }, [id]);
  const { data, isLoading: regsLoading } = useEventRegistrations(
    id,
    { limit: REG_PAGE_SIZE, offset: regPage * REG_PAGE_SIZE },
    !!id,
  );
  const regs = data?.registrations ?? [];
  const regTotal = data?.total ?? 0;
  const regPageCount = Math.max(1, Math.ceil(regTotal / REG_PAGE_SIZE));

  const statusMut = useUpdateRegistrationStatus();
  const approveMut = useApproveRegistration();
  const rejectMut = useRejectRegistration();
  const checkInMut = useCheckIn();
  const checkInByTokenMut = useCheckInByEventToken(id);
  const [scannerOpen, setScannerOpen] = useState(false);

  const handleScannedToken = useCallback(
    (token: string) => {
      checkInByTokenMut.mutate({ access_token: token });
    },
    [checkInByTokenMut],
  );

  useEffect(() => {
    document.title = event ? `${event.title} — Manage — EnviToo` : "Manage — EnviToo";
  }, [event]);

  const eventLoading = myEventsLoading || publicLoading;

  const eventUrl = event ? `${window.location.origin}/e/${event.slug}` : "";
  const [copied, setCopied] = useState(false);

  const copyLink = useCallback(async () => {
    if (!eventUrl) return;
    try {
      await navigator.clipboard.writeText(eventUrl);
      setCopied(true);
      toast.success("Link copied!");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Could not copy link");
    }
  }, [eventUrl]);

  if (eventLoading && !event) {
    return <EventDashboardSkeleton />;
  }

  if (!event || (!isOrganizer && !isVolunteerAdmin)) {
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
  const activeRegistrations = event.registration_count;
  const seatsLeft = Math.max(0, event.capacity - activeRegistrations);
  const canModerateRegistrations = isVolunteerAdmin && event.requires_approval;
  const moderationPending =
    statusMut.isPending || approveMut.isPending || rejectMut.isPending;

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
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:flex-wrap">
            <Button asChild variant="outline" className="w-full sm:w-auto">
              <Link to={`/e/${event.slug}`} target="_blank">
                View public page
              </Link>
            </Button>
            <div className="flex gap-2">
              <Button variant="outline" size="icon" onClick={copyLink} title="Copy link" className="flex-1 sm:flex-initial">
                {copied ? <CheckCircle2 className="size-4 text-primary" /> : <Copy className="size-4" />}
              </Button>
              <Button asChild variant="outline" className="flex-1 sm:flex-initial">
                <Link to={`/manage/${event.id}/edit`}>
                  <Pencil data-icon="inline-start" />
                  Edit details
                </Link>
              </Button>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="mt-8">
          <Tabs defaultValue="registrations">
            <TabsList>
              <TabsTrigger value="registrations">
                <Users data-icon="inline-start" />
                Registrations
              </TabsTrigger>
              {(isOrganizer || isVolunteerAdmin) && (
                <TabsTrigger value="volunteers">
                  <HeartHandshake data-icon="inline-start" />
                  Volunteers
                </TabsTrigger>
              )}
            </TabsList>

            <TabsContent value="registrations">
              {/* Stats */}
              <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <Stat icon={<Users className="size-5" />} label="Capacity" value={event.capacity} />
                <Stat icon={<CheckCircle2 className="size-5" />} label="Approved" value={approved} />
                <Stat icon={<Users className="size-5" />} label="Pending" value={pending} />
                <Stat icon={<CheckCircle2 className="size-5" />} label="Checked in" value={checkedIn} />
              </div>
              {event.status !== "cancelled" && (
                <p className="mt-3 text-sm text-muted-foreground">{seatsLeft} seats remaining.</p>
              )}

              {/* Registrations */}
              <div className="mt-8">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h2 className="text-xl font-semibold">Registrations</h2>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {regTotal} total
                      {regPageCount > 1 && (
                        <span className="text-muted-foreground/80">
                          {" "}
                          · page {regPage + 1} of {regPageCount}
                        </span>
                      )}
                    </p>
                  </div>
                  {(isOrganizer || isVolunteerAdmin) && (
                    <Button variant="outline" size="sm" onClick={() => setScannerOpen(true)} className="gap-1.5">
                      <ScanLine className="size-4" />
                      Scan to check in
                    </Button>
                  )}
                </div>

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
                ) : regTotal === 0 ? (
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
                  <>
                    {/* Desktop: table */}
                    <div className="mt-6 hidden overflow-x-auto rounded-2xl border md:block">
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
                              isOrganizer={isOrganizer}
                              canModerate={canModerateRegistrations}
                              statusPending={moderationPending}
                              checkInPending={checkInMut.isPending || checkInByTokenMut.isPending}
                              onStatus={(status) => statusMut.mutate({ id: reg.id, values: { status } })}
                              onApprove={() => approveMut.mutate(reg.id)}
                              onReject={() => rejectMut.mutate(reg.id)}
                              onCheckIn={() =>
                                reg.access_token &&
                                checkInMut.mutate({ id: reg.id, values: { access_token: reg.access_token } })
                              }
                            />
                          ))}
                        </TableBody>
                      </Table>
                    </div>

                    {/* Mobile: stacked cards */}
                    <div className="mt-6 flex flex-col gap-3 md:hidden">
                      {regs.map((reg) => (
                        <RegistrationCard
                          key={reg.id}
                          reg={reg}
                          eventCancelled={event.status === "cancelled"}
                          isOrganizer={isOrganizer}
                          canModerate={canModerateRegistrations}
                          statusPending={moderationPending}
                          checkInPending={checkInMut.isPending || checkInByTokenMut.isPending}
                          onStatus={(status) => statusMut.mutate({ id: reg.id, values: { status } })}
                          onApprove={() => approveMut.mutate(reg.id)}
                          onReject={() => rejectMut.mutate(reg.id)}
                          onCheckIn={() =>
                            reg.access_token &&
                            checkInMut.mutate({ id: reg.id, values: { access_token: reg.access_token } })
                          }
                        />
                      ))}
                    </div>

                    {regPageCount > 1 && (
                      <div className="mt-4 flex items-center justify-between gap-3">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={regPage === 0}
                          onClick={() => setRegPage((p) => Math.max(0, p - 1))}
                        >
                          Previous
                        </Button>
                        <span className="text-sm text-muted-foreground">
                          Page {regPage + 1} of {regPageCount}
                        </span>
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={regPage >= regPageCount - 1}
                          onClick={() => setRegPage((p) => Math.min(regPageCount - 1, p + 1))}
                        >
                          Next
                        </Button>
                      </div>
                    )}
                  </>
                )}
              </div>
            </TabsContent>

            {(isOrganizer || isVolunteerAdmin) && (
              <TabsContent value="volunteers">
                <VolunteerDashboard event={event} />
              </TabsContent>
            )}
          </Tabs>
        </div>
      </PageContainer>

      <CheckInScanner
        open={scannerOpen}
        onOpenChange={setScannerOpen}
        onResult={handleScannedToken}
      />
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
  isOrganizer,
  canModerate,
  statusPending,
  checkInPending,
  onStatus,
  onApprove,
  onReject,
  onCheckIn,
}: {
  reg: Registration;
  eventCancelled: boolean;
  isOrganizer: boolean;
  canModerate: boolean;
  statusPending: boolean;
  checkInPending: boolean;
  onStatus: (status: UpdateRegistrationStatusValues["status"]) => void;
  onApprove: () => void;
  onReject: () => void;
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
          {isOrganizer && (
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
          )}
          {canModerate && reg.status === "pending" && (
            <>
              <Button
                size="xs"
                className="bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                onClick={onApprove}
                disabled={busy}
              >
                <Check className="size-3.5" /> Approve
              </Button>
              <Button
                size="xs"
                className="bg-rose-50 text-rose-700 hover:bg-rose-100"
                onClick={onReject}
                disabled={busy}
              >
                <X className="size-3.5" /> Reject
              </Button>
            </>
          )}
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

function RegistrationCard({
  reg,
  eventCancelled,
  isOrganizer,
  canModerate,
  statusPending,
  checkInPending,
  onStatus,
  onApprove,
  onReject,
  onCheckIn,
}: {
  reg: Registration;
  eventCancelled: boolean;
  isOrganizer: boolean;
  canModerate: boolean;
  statusPending: boolean;
  checkInPending: boolean;
  onStatus: (status: UpdateRegistrationStatusValues["status"]) => void;
  onApprove: () => void;
  onReject: () => void;
  onCheckIn: () => void;
}) {
  const busy = statusPending || checkInPending;

  function handleStatusChange(next: string) {
    const parsed = UpdateRegistrationStatusInput.safeParse({ status: next });
    if (!parsed.success || parsed.data.status === reg.status) return;
    onStatus(parsed.data.status);
  }

  return (
    <div className="rounded-2xl border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate font-medium">{reg.name}</p>
          <p className="truncate text-sm text-muted-foreground">{reg.email}</p>
        </div>
        <RegStatusBadge status={reg.status} />
      </div>
      <div className="mt-3 flex items-center gap-2 text-sm">
        {reg.checked_in_at ? (
          <span className="inline-flex items-center gap-1.5 text-emerald-700">
            <CheckCircle2 className="size-4" /> Checked in
          </span>
        ) : (
          <span className="text-muted-foreground">Not checked in</span>
        )}
      </div>
      <div className="mt-3 flex flex-col gap-2">
        {isOrganizer && (
          <Select value={reg.status} onValueChange={handleStatusChange} disabled={busy}>
            <SelectTrigger size="sm" className="w-full capitalize">
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
        )}
        {canModerate && reg.status === "pending" && (
          <div className="flex gap-2">
            <Button
              size="sm"
              className="flex-1 bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
              onClick={onApprove}
              disabled={busy}
            >
              <Check className="size-4" /> Approve
            </Button>
            <Button
              size="sm"
              className="flex-1 bg-rose-50 text-rose-700 hover:bg-rose-100"
              onClick={onReject}
              disabled={busy}
            >
              <X className="size-4" /> Reject
            </Button>
          </div>
        )}
        {reg.status === "approved" && !reg.checked_in_at && !eventCancelled && (
          <Button size="sm" onClick={onCheckIn} disabled={busy} className="w-full">
            {checkInPending ? <Spinner /> : "Check in"}
          </Button>
        )}
      </div>
    </div>
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
