import { ShieldAlert } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { VolunteerStats } from "@/components/domain/VolunteerStats";
import { VolunteerSettingsPanel } from "@/components/domain/VolunteerSettingsPanel";
import { VolunteerRolesPanel } from "@/components/domain/VolunteerRolesPanel";
import { VolunteerApplicationsPanel } from "@/components/domain/VolunteerApplicationsPanel";
import { VolunteerRosterPanel } from "@/components/domain/VolunteerRosterPanel";
import { useMe } from "@/hooks/use-me";
import {
  useVolunteerSettings,
  useEventVolunteerApplications,
  useEventVolunteers,
  useMyEventStaff,
} from "@/hooks/use-volunteers";
import type { Event } from "@/lib/api";

export function VolunteerDashboard({ event }: { event: Event }) {
  const { data: me } = useMe();
  const { data: settings } = useVolunteerSettings(event.id);
  const { data: appsData } = useEventVolunteerApplications(event.id, { limit: 100 });
  const { data: volData } = useEventVolunteers(event.id, { limit: 100 });
  const { data: staffData } = useMyEventStaff({});

  const isOrganizer = !!me && event.organizer_id === me.id;
  const isVolunteerAdmin =
    !isOrganizer && staffData?.events.some((e) => e.event_id === event.id) === true;

  if (!me) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-24 rounded-2xl" />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <VolunteerStats
        applications={appsData?.applications ?? []}
        volunteers={volData ?? { volunteers: [], limit: 0, offset: 0, total: 0 }}
      />

      {isVolunteerAdmin && (
        <Alert>
          <ShieldAlert className="size-4" />
          <AlertTitle>Volunteer admin access</AlertTitle>
          <AlertDescription>
            You can review applications and view the volunteer roster. Settings and roles are
            managed by the organizer.
          </AlertDescription>
        </Alert>
      )}

      {isOrganizer && (
        <>
          <VolunteerSettingsPanel eventId={event.id} settings={settings} />
          <VolunteerRolesPanel eventId={event.id} />
        </>
      )}

      <VolunteerApplicationsPanel
        eventId={event.id}
        isOrganizer={isOrganizer}
        applications={appsData?.applications ?? []}
        total={appsData?.total ?? 0}
      />
      <VolunteerRosterPanel
        volunteers={volData?.volunteers ?? []}
        total={volData?.total ?? 0}
      />
    </div>
  );
}
