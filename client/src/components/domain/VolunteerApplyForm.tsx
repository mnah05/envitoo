import { useState } from "react";
import { Link } from "react-router-dom";
import { useUser } from "@clerk/clerk-react";
import { CheckCircle2, HeartHandshake, LogIn } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { RoleBadge } from "@/components/domain/RoleBadge";
import { useApplyVolunteer, useVolunteerRoles } from "@/hooks/use-volunteers";
import { CreateVolunteerApplicationInput, type Event, type VolunteerSettings } from "@/lib/api";
import { ApiError } from "@/lib/api";

export function VolunteerApplyForm({
  event,
  settings,
}: {
  event: Event;
  settings: VolunteerSettings;
}) {
  const { isSignedIn, user } = useUser();
  const { data: roles } = useVolunteerRoles(event.id);
  const applyMut = useApplyVolunteer(event.id);

  const defaultName = isSignedIn && user ? user.username ?? user.fullName ?? "" : "";
  const defaultEmail = isSignedIn && user ? user.primaryEmailAddress?.emailAddress ?? "" : "";

  const [roleId, setRoleId] = useState("");
  const [name, setName] = useState(defaultName);
  const [email, setEmail] = useState(defaultEmail);
  const [phone, setPhone] = useState("");
  const [description, setDescription] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [done, setDone] = useState(false);
  const [alertMsg, setAlertMsg] = useState("");

  if (!settings.enabled) return null;

  if (!isSignedIn) {
    return (
      <div className="rounded-2xl border bg-card p-6 shadow-sm">
        <div className="flex flex-col items-center gap-3 text-center">
          <div className="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <HeartHandshake className="size-5" />
          </div>
          <div>
            <p className="display-headline text-[1.0625rem]">Volunteer at this event</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Help the team and get behind the scenes.
            </p>
          </div>
          <Button asChild className="mt-1 w-full">
            <Link to="/signin">
              <LogIn data-icon="inline-start" />
              Sign in to volunteer
            </Link>
          </Button>
        </div>
        {roles && roles.length > 0 && (
          <RolesList roles={roles} />
        )}
      </div>
    );
  }

  if (done) {
    return (
      <div className="rounded-2xl border bg-card p-6 shadow-sm">
        <div className="flex flex-col items-center gap-3 rounded-xl bg-emerald-50/60 p-6 text-center">
          <CheckCircle2 className="size-8 text-emerald-600" />
          <div>
            <p className="display-headline text-[1.0625rem] text-foreground">Application submitted!</p>
            <p className="mt-1 text-sm text-muted-foreground">
              We'll review your application and notify you.
            </p>
          </div>
          <Button asChild variant="outline" className="mt-2 w-full">
            <Link to="/home">View my applications</Link>
          </Button>
        </div>
      </div>
    );
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (applyMut.isPending) return;

    const errs: Record<string, string> = {};
    if (!roleId) errs.role_id = "Please select a role";
    if (!description.trim()) errs.description = "Tell the organizer why you'd be a great fit";
    if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errs.email = "Enter a valid email";
    setErrors({});
    setAlertMsg("");

    const parsed = CreateVolunteerApplicationInput.safeParse({
      role_id: roleId,
      name: name || undefined,
      email: email || undefined,
      phone: phone || undefined,
      description: description.trim(),
    });
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        errs[issue.path[0] as string] = errs[issue.path[0] as string] ?? issue.message;
      }
      setErrors(errs);
      return;
    }

    try {
      await applyMut.mutateAsync(parsed.data);
      setDone(true);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setAlertMsg(err.message);
      } else {
        setAlertMsg("Something went wrong. Please try again.");
      }
    }
  }

  return (
    <div className="rounded-2xl border bg-card p-6 shadow-sm">
      <div className="flex items-center gap-2">
        <HeartHandshake className="size-5 text-primary" />
        <h3 className="display-headline text-[1.0625rem]">Volunteer at this event</h3>
      </div>

      {settings.description && (
        <p className="mt-2 text-sm text-muted-foreground">{settings.description}</p>
      )}

      <Separator className="my-4" />

      {alertMsg && (
        <Alert variant="destructive" className="mb-4">
          <AlertDescription>{alertMsg}</AlertDescription>
        </Alert>
      )}

      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <FieldGroup>
          <Field data-invalid={!!errors.role_id}>
            <FieldLabel htmlFor="volunteer-role">Role</FieldLabel>
            <Select value={roleId} onValueChange={setRoleId}>
              <SelectTrigger id="volunteer-role">
                <SelectValue placeholder="Choose a role" />
              </SelectTrigger>
              <SelectContent>
                {roles?.map((role) => (
                  <SelectItem key={role.id} value={role.id}>
                    {role.name}
                    {role.is_admin && " · Admin"}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.role_id && <FieldError>{errors.role_id}</FieldError>}
          </Field>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="volunteer-name">Name</FieldLabel>
              <Input
                id="volunteer-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={defaultName || "Your name"}
                autoComplete="name"
              />
            </Field>
            <Field data-invalid={!!errors.email}>
              <FieldLabel htmlFor="volunteer-email">Email</FieldLabel>
              <Input
                id="volunteer-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder={defaultEmail || "you@example.com"}
                autoComplete="email"
              />
              {errors.email && <FieldError>{errors.email}</FieldError>}
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor="volunteer-phone">Phone (optional)</FieldLabel>
            <Input
              id="volunteer-phone"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="+1 234 567 890"
              autoComplete="tel"
            />
          </Field>

          <Field data-invalid={!!errors.description}>
            <FieldLabel htmlFor="volunteer-description">Why you're a great fit</FieldLabel>
            <Textarea
              id="volunteer-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Tell the organizer about your experience and why you want to volunteer"
              rows={4}
            />
            {errors.description ? (
              <FieldError>{errors.description}</FieldError>
            ) : (
              <FieldDescription>Up to 5000 characters.</FieldDescription>
            )}
          </Field>
        </FieldGroup>

        <Button type="submit" size="lg" className="w-full" disabled={applyMut.isPending}>
          {applyMut.isPending && <Spinner data-icon="inline-start" />}
          Apply to volunteer
        </Button>
        <p className="text-center text-xs text-muted-foreground">
          We'll review your application and get back to you.
        </p>
      </form>

      {roles && roles.length > 0 && (
        <RolesList roles={roles} />
      )}
    </div>
  );
}

function RolesList({ roles }: { roles: { name: string; is_admin: boolean }[] }) {
  return (
    <div className="mt-4 border-t pt-4">
      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        Roles available
      </p>
      <div className="flex flex-wrap gap-2">
        {roles.map((role) => (
          <span
            key={role.name}
            className="inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm"
          >
            {role.name}
            {role.is_admin && <RoleBadge isAdmin />}
          </span>
        ))}
      </div>
    </div>
  );
}
