import { Check, ClipboardList, X } from "lucide-react";
import { Button } from "@/components/ui/button";
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
import { Skeleton } from "@/components/ui/skeleton";
import { RoleBadge } from "@/components/domain/RoleBadge";
import { VolunteerAppStatusBadge } from "@/components/domain/StatusBadge";
import {
  useEventVolunteerApplications,
  useApproveVolunteerApplication,
  useRejectVolunteerApplication,
} from "@/hooks/use-volunteers";
import { formatCardDate } from "@/lib/format";
import type { VolunteerApplication } from "@/lib/api";

const PAGE_SIZE = 20;

export function VolunteerApplicationsPanel({
  eventId,
  isOrganizer,
}: {
  eventId: string;
  isOrganizer: boolean;
}) {
  const { data, isLoading } = useEventVolunteerApplications(eventId, { limit: PAGE_SIZE });
  const apps = data?.applications ?? [];
  const total = data?.total ?? 0;
  const approveMut = useApproveVolunteerApplication(eventId);
  const rejectMut = useRejectVolunteerApplication(eventId);

  const pending = approveMut.isPending || rejectMut.isPending;

  return (
    <div className="rounded-2xl border bg-card p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ClipboardList className="size-5 text-muted-foreground" />
          <h3 className="font-semibold">Applications</h3>
        </div>
        {total > 0 && (
          <span className="text-sm text-muted-foreground">
            {total} total · {apps.filter((a) => a.status === "pending").length} pending
          </span>
        )}
      </div>

      <div className="mt-4">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        ) : apps.length === 0 ? (
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
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </div>
  );
}

function ApplicationRow({
  app,
  isOrganizer,
  pending,
  onApprove,
  onReject,
}: {
  app: VolunteerApplication;
  isOrganizer: boolean;
  pending: boolean;
  onApprove: () => void;
  onReject: () => void;
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
        {isPending ? (
          <div className="flex items-center justify-end gap-1.5">
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
          </div>
        ) : (
          <span className="text-sm text-muted-foreground">—</span>
        )}
      </TableCell>
    </TableRow>
  );
}
