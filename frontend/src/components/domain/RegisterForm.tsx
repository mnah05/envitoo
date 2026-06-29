import { useState } from "react";
import { Link } from "react-router-dom";
import { useUser } from "@clerk/clerk-react";
import { CheckCircle2, Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Spinner } from "@/components/ui/spinner";
import { useCreateGuestRegistration, useCreateRegistration } from "@/hooks/use-registrations";
import { CreateGuestRegistrationInput, type Event } from "@/lib/api";
import { ApiError } from "@/lib/api";

export function RegisterForm({ event }: { event: Event }) {
  const { isSignedIn, user } = useUser();
  const guestMut = useCreateGuestRegistration();
  const userMut = useCreateRegistration();

  const defaultName = isSignedIn && user ? user.username ?? user.fullName ?? "" : "";
  const defaultEmail = isSignedIn && user ? user.primaryEmailAddress?.emailAddress ?? "" : "";

  const [name, setName] = useState<string>(defaultName);
  const [email, setEmail] = useState<string>(defaultEmail);
  const [errors, setErrors] = useState<{ name?: string; email?: string }>({});
  const [done, setDone] = useState<{ token?: string } | null>(null);

  const pending = guestMut.isPending || userMut.isPending;

  function validate(): boolean {
    const e: typeof errors = {};
    if (!isSignedIn) {
      if (!name.trim()) e.name = "Name is required";
      if (!email.trim()) e.email = "Email is required";
      else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) e.email = "Enter a valid email";
    } else {
      if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) e.email = "Enter a valid email";
    }
    setErrors(e);
    return Object.keys(e).length === 0;
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (pending) return;
    if (!validate()) return;

    try {
      if (isSignedIn) {
        await userMut.mutateAsync({
          eventId: event.id,
          values: {
            name: name || undefined,
            email: email || undefined,
          },
        });
        setDone({});
      } else {
        const parsed = CreateGuestRegistrationInput.safeParse({ name, email });
        if (!parsed.success) {
          setErrors({ name: parsed.error.issues[0]?.message });
          return;
        }
        const reg = await guestMut.mutateAsync({ eventId: event.id, values: parsed.data });
        setDone({ token: reg.access_token });
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setErrors({ email: err.message });
      }
    }
  }

  if (done) {
    return (
      <div className="flex flex-col items-center gap-3 rounded-2xl border bg-emerald-50/60 p-6 text-center">
        <CheckCircle2 className="size-8 text-emerald-600" />
        <div>
          <p className="font-semibold text-foreground">You're registered!</p>
          <p className="mt-1 text-sm text-muted-foreground">
            {event.requires_approval
              ? "We'll review your registration and confirm by email."
              : "We've sent the details to your inbox."}
          </p>
        </div>
        {done.token ? (
          <Button asChild className="mt-2 w-full">
            <Link to={`/ticket?token=${done.token}`}>
              <Ticket data-icon="inline-start" />
              View your ticket
            </Link>
          </Button>
        ) : (
          <Button asChild variant="outline" className="mt-2 w-full">
            <Link to="/home">View my tickets</Link>
          </Button>
        )}
      </div>
    );
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-5">
      <FieldGroup>
        {!isSignedIn && (
          <>
            <Field data-invalid={!!errors.name}>
              <FieldLabel htmlFor="reg-name">Name</FieldLabel>
              <Input
                id="reg-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Your name"
                aria-invalid={!!errors.name}
                autoComplete="name"
              />
              {errors.name && <FieldError>{errors.name}</FieldError>}
            </Field>
            <Field data-invalid={!!errors.email}>
              <FieldLabel htmlFor="reg-email">Email</FieldLabel>
              <Input
                id="reg-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                aria-invalid={!!errors.email}
                autoComplete="email"
              />
              {errors.email ? (
                <FieldError>{errors.email}</FieldError>
              ) : (
                <FieldDescription>We'll send your ticket and updates here.</FieldDescription>
              )}
            </Field>
          </>
        )}

        {isSignedIn && (
          <Field data-invalid={!!errors.email}>
            <FieldLabel htmlFor="reg-email">Email (optional)</FieldLabel>
            <Input
              id="reg-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={defaultEmail}
              aria-invalid={!!errors.email}
              autoComplete="email"
            />
            {errors.email && <FieldError>{errors.email}</FieldError>}
            <FieldDescription>
              Defaults to your account email. Override it for this event if you like.
            </FieldDescription>
          </Field>
        )}
      </FieldGroup>

      <Button type="submit" size="lg" className="w-full" disabled={pending}>
        {pending && <Spinner data-icon="inline-start" />}
        {event.requires_approval ? "Request to register" : "Get tickets"}
      </Button>
      <p className="text-center text-xs text-muted-foreground">
        {isSignedIn ? "You'll register with your Pulse account." : "No account needed — register as a guest."}
      </p>
    </form>
  );
}
