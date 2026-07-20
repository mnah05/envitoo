import { useState } from "react";
import { Check, ClipboardList, Eye, Mail, Phone, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import { RoleBadge } from "@/components/domain/RoleBadge";
import { VolunteerAppStatusBadge } from "@/components/domain/StatusBadge";
import {
  useApproveVolunteerApplication,
  useRejectVolunteerApplication,
} from "@/hooks/use-volunteers";
import { formatCardDate } from "@/lib/format";
import type { VolunteerApplication } from "@/lib/api";

export function VolunteerApplicationsPanel({
  eventId,
  isOrganizer,
  applications: apps,
  total,
}: {
  eventId: string;
  isOrganizer: boolean;
  applications: VolunteerApplication[];
  total: number;
}) {
  const approveMut = useApproveVolunteerApplication(eventId);
  const rejectMut = useRejectVolunteerApplication(eventId);
  const [selected, setSelected] = useState<VolunteerApplication | null>(null);

  const pending = approveMut.isPending || rejectMut.isPending;

  function handleModerate(action: "approve" | "reject") {
    if (!selected) return;
    if (action === "approve") approveMut.mutate(selected.id);
    else rejectMut.mutate(selected.id);
    setSelected(null);
  }

  return (
    <div className="rounded-2xl border bg-card p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ClipboardList className="size-5 text-muted-foreground" />
          <h3 className="display-headline text-[1.0625rem]">Applications</h3>
        </div>
        {total > 0 && (
          <span className="text-sm text-muted-foreground">
            {total} total · {apps.filter((a) => a.status === "pending").length} pending
          </span>
        )}
      </div>

      <div className="mt-4">
        {apps.length === 0 ? (
          <Empty className="border">
            <EmptyMedia variant="icon">
              <ClipboardList className="size-6" />
            </EmptyMedia>
            <EmptyTitle>No applications yet</EmptyTitle>
            <EmptyDescription>
              Share your event to attract volunteers.
            </EmptyDescription>
          </Empty>
        ) : (
          <div className="overflow-x-auto rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Applicant</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Applied</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apps.map((app) => (
                  <ApplicationRow
                    key={app.id}
                    app={app}
                    isOrganizer={isOrganizer}
                    pending={pending}
                    onApprove={() => approveMut.mutate(app.id)}
                    onReject={() => rejectMut.mutate(app.id)}
                    onView={() => setSelected(app)}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {/* Application detail dialog */}
      <Dialog open={selected !== null} onOpenChange={(o) => !o && setSelected(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{selected?.name}</DialogTitle>
            <DialogDescription className="flex flex-wrap items-center gap-2">
              {selected && (
                <>
                  <span className="inline-flex items-center gap-1.5">
                    {selected.role_name}
                    {selected.is_admin && <RoleBadge isAdmin />}
                  </span>
                  <span aria-hidden>·</span>
                  <VolunteerAppStatusBadge status={selected.status} />
                  <span aria-hidden>·</span>
                  <span>Applied {formatCardDate(selected.created_at)}</span>
                </>
              )}
            </DialogDescription>
          </DialogHeader>

          {selected && (
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-2 text-sm">
                <a
                  href={`mailto:${selected.email}`}
                  className="inline-flex items-center gap-2 text-primary hover:underline"
                >
                  <Mail className="size-4" /> {selected.email}
                </a>
                {selected.phone && (
                  <a
                    href={`tel:${selected.phone}`}
                    className="inline-flex items-center gap-2 text-muted-foreground hover:text-foreground"
                  >
                    <Phone className="size-4" /> {selected.phone}
                  </a>
                )}
              </div>
              <div>
                <p className="mb-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  Why they're a great fit
                </p>
                <p className="max-h-[40vh] overflow-y-auto whitespace-pre-wrap text-pretty leading-relaxed">
                  {selected.description}
                </p>
              </div>
              {selected.status === "pending" && (
                <div className="flex items-center gap-2 pt-1">
                  <Button
                    size="sm"
                    className="bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                    onClick={() => handleModerate("approve")}
                    disabled={pending || (!isOrganizer && selected.is_admin)}
                  >
                    <Check className="size-4" /> Approve
                  </Button>
                  <Button
                    size="sm"
                    className="bg-rose-50 text-rose-700 hover:bg-rose-100"
                    onClick={() => handleModerate("reject")}
                    disabled={pending || (!isOrganizer && selected.is_admin)}
                  >
                    <X className="size-4" /> Reject
                  </Button>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ApplicationRow({
  app,
  isOrganizer,
  pending,
  onApprove,
  onReject,
  onView,
}: {
  app: VolunteerApplication;
  isOrganizer: boolean;
  pending: boolean;
  onApprove: () => void;
  onReject: () => void;
  onView: () => void;
}) {
  const isPending = app.status === "pending";
  const adminRestricted = !isOrganizer && app.is_admin;

  return (
    <TableRow>
      <TableCell>
        <div className="flex flex-col">
          <span className="font-medium">{app.name}</span>
          <span className="text-xs text-muted-foreground">{app.email}</span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-1.5">
          <span>{app.role_name}</span>
          {app.is_admin && <RoleBadge isAdmin />}
        </div>
      </TableCell>
      <TableCell>
        <VolunteerAppStatusBadge status={app.status} />
      </TableCell>
      <TableCell className="text-muted-foreground">{formatCardDate(app.created_at)}</TableCell>
      <TableCell className="text-right">
        <div className="flex items-center justify-end gap-1.5">
          <Button size="xs" variant="outline" onClick={onView} aria-label="View application">
            <Eye className="size-3" />
            View
          </Button>
          {isPending && (
            <Button
              size="xs"
              className="bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
              onClick={onApprove}
              disabled={pending || adminRestricted}
              title={adminRestricted ? "Only the organizer can moderate admin-role applications" : undefined}
            >
              <Check className="size-3" />
              Approve
            </Button>
          )}
          {isPending && (
            <Button
              size="xs"
              className="bg-rose-50 text-rose-700 hover:bg-rose-100"
              onClick={onReject}
              disabled={pending || adminRestricted}
              title={adminRestricted ? "Only the organizer can moderate admin-role applications" : undefined}
            >
              <X className="size-3" />
              Reject
            </Button>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}
