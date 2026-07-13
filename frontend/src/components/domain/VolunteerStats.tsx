import { CheckCircle2, ClipboardList, Clock, HeartHandshake } from "lucide-react";
import type { VolunteerApplication, VolunteerList } from "@/lib/api";

export function VolunteerStats({
  applications,
  volunteers,
}: {
  applications: VolunteerApplication[];
  volunteers: VolunteerList;
}) {
  const pending = applications.filter((a) => a.status === "pending").length;
  const approved = applications.filter((a) => a.status === "approved").length;
  const roster = volunteers.total;

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <Stat icon={<ClipboardList className="size-5" />} label="Applied" value={applications.length} />
      <Stat icon={<Clock className="size-5" />} label="Pending" value={pending} />
      <Stat icon={<CheckCircle2 className="size-5" />} label="Approved" value={approved} />
      <Stat icon={<HeartHandshake className="size-5" />} label="Roster" value={roster} />
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
        <span className="text-eyebrow">{label}</span>
      </div>
      <p className="text-stat mt-2 text-2xl">{value}</p>
    </div>
  );
}
