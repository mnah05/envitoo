import { useState } from "react";
import { Plus, Shield, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Spinner } from "@/components/ui/spinner";
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
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { useVolunteerRoles, useCreateVolunteerRole, useDeleteVolunteerRole } from "@/hooks/use-volunteers";
import { formatCardDate } from "@/lib/format";
import { ApiError } from "@/lib/api";

export function VolunteerRolesPanel({ eventId }: { eventId: string }) {
  const { data: roles, isLoading } = useVolunteerRoles(eventId);
  const createMut = useCreateVolunteerRole(eventId);
  const deleteMut = useDeleteVolunteerRole(eventId);

  const [newName, setNewName] = useState("");
  const [newIsAdmin, setNewIsAdmin] = useState(false);
  const [createError, setCreateError] = useState("");

  function handleAdd() {
    if (!newName.trim()) return;
    setCreateError("");
    createMut.mutate(
      { name: newName.trim(), is_admin: newIsAdmin },
      {
        onSuccess: () => {
          setNewName("");
          setNewIsAdmin(false);
        },
        onError: (err) => {
          if (err instanceof ApiError && err.status === 409) {
            setCreateError("Role name already exists");
          }
        },
      },
    );
  }

  return (
    <div className="rounded-2xl border bg-card p-6 shadow-sm">
      <div className="flex items-center gap-2">
        <Shield className="size-5 text-muted-foreground" />
        <h3 className="display-headline text-[1.0625rem]">Roles</h3>
      </div>
      <p className="mt-1 text-sm text-muted-foreground">
        Define the roles volunteers can apply for.
      </p>

      <div className="mt-4">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full rounded-lg" />
            ))}
          </div>
        ) : !roles || roles.length === 0 ? (
          <Empty className="border">
            <EmptyMedia variant="icon">
              <Shield className="size-6" />
            </EmptyMedia>
            <EmptyTitle>No roles yet</EmptyTitle>
            <EmptyDescription>Add a role to start accepting volunteer applications.</EmptyDescription>
          </Empty>
        ) : (
          <div className="overflow-x-auto rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {roles.map((role) => (
                  <TableRow key={role.id}>
                    <TableCell className="font-medium">{role.name}</TableCell>
                    <TableCell>
                      {role.is_admin ? <RoleBadge isAdmin /> : <span className="text-sm text-muted-foreground">Volunteer</span>}
                    </TableCell>
                    <TableCell className="text-muted-foreground">{formatCardDate(role.created_at)}</TableCell>
                    <TableCell className="text-right">
                      <DeleteRoleDialog
                        roleName={role.name}
                        pending={deleteMut.isPending}
                        onConfirm={() => deleteMut.mutate(role.id)}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      <div className="mt-4 flex flex-col gap-3 rounded-xl border bg-muted/40 p-4 sm:flex-row sm:items-end">
        <div className="flex-1">
          <Input
            value={newName}
            onChange={(e) => {
              setNewName(e.target.value);
              setCreateError("");
            }}
            placeholder="Role name (e.g. Photographer)"
            aria-invalid={!!createError}
          />
          {createError && <p className="mt-1 text-xs text-destructive">{createError}</p>}
        </div>
        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          <Switch checked={newIsAdmin} onCheckedChange={setNewIsAdmin} />
          Admin role
        </label>
        <Button onClick={handleAdd} size="sm" disabled={createMut.isPending || !newName.trim()}>
          {createMut.isPending ? <Spinner /> : <Plus data-icon="inline-start" />}
          Add role
        </Button>
      </div>
    </div>
  );
}

function DeleteRoleDialog({
  roleName,
  pending,
  onConfirm,
}: {
  roleName: string;
  pending: boolean;
  onConfirm: () => void;
}) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-destructive">
          <Trash2 className="size-4" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-destructive/10 text-destructive">
            <Trash2 />
          </AlertDialogMedia>
          <AlertDialogTitle>Delete role "{roleName}"?</AlertDialogTitle>
          <AlertDialogDescription>
            This will remove the role from the event. If any volunteers have applied for this role,
            it cannot be deleted until those applications are resolved.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={onConfirm}>
            {pending ? "Deleting…" : "Delete role"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
