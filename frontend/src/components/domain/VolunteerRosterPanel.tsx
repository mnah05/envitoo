import { HeartHandshake } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
import { useEventVolunteers } from "@/hooks/use-volunteers";
import { formatCardDate, initials } from "@/lib/format";

export function VolunteerRosterPanel({ eventId }: { eventId: string }) {
  const { data, isLoading } = useEventVolunteers(eventId);
  const volunteers = data?.volunteers ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="rounded-2xl border bg-card p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <HeartHandshake className="size-5 text-muted-foreground" />
          <h3 className="font-semibold">Confirmed volunteers</h3>
        </div>
        {total > 0 && (
          <span className="text-sm text-muted-foreground">{total} volunteers</span>
        )}
      </div>

      <div className="mt-4">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        ) : volunteers.length === 0 ? (
          <Empty className="border">
            <EmptyMedia variant="icon">
              <HeartHandshake className="size-6" />
            </EmptyMedia>
            <EmptyTitle>No approved volunteers yet</EmptyTitle>
            <EmptyDescription>
              Applications will appear here once approved.
            </EmptyDescription>
          </Empty>
        ) : (
          <div className="overflow-x-auto rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Volunteer</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Joined</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {volunteers.map((vol) => (
                  <TableRow key={vol.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Avatar size="sm">
                          {vol.avatar_url && <AvatarImage src={vol.avatar_url} alt="" />}
                          <AvatarFallback>
                            {initials(vol.username ?? vol.email)}
                          </AvatarFallback>
                        </Avatar>
                        <div className="flex flex-col">
                          <span className="font-medium">{vol.username ?? vol.email}</span>
                          <span className="text-xs text-muted-foreground">{vol.email}</span>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm">{vol.role}</span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{formatCardDate(vol.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </div>
  );
}
