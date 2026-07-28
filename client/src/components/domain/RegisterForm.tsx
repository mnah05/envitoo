import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useUser } from "@clerk/clerk-react";
import { CheckCircle2, Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Spinner } from "@/components/ui/spinner";
import {
  useCreateGuestRegistration,
  useCreateRegistration,
  useRequestGuestOTP,
  useVerifyGuestOTP,
} from "@/hooks/use-registrations";
import {
  CreateGuestRegistrationInput,
  RequestGuestOTPInput,
  VerifyGuestOTPInput,
  type Event,
} from "@/lib/api";
import { ApiError } from "@/lib/api";

type GuestStep = "details" | "code";

const RESEND_COOLDOWN_SECONDS = 60;

export function RegisterForm({ event }: { event: Event }) {
  const { isSignedIn, user } = useUser();
  const guestMut = useCreateGuestRegistration();
  const userMut = useCreateRegistration();
  const requestOtpMut = useRequestGuestOTP();
  const verifyOtpMut = useVerifyGuestOTP();

  const defaultName = isSignedIn && user ? user.username ?? user.fullName ?? "" : "";
  const defaultEmail = isSignedIn && user ? user.primaryEmailAddress?.emailAddress ?? "" : "";

  const [name, setName] = useState<string>(defaultName);
  const [email, setEmail] = useState<string>(defaultEmail);
  const [otp, setOtp] = useState("");
  const [guestStep, setGuestStep] = useState<GuestStep>("details");
  const [cooldownLeft, setCooldownLeft] = useState(0);
  const [errors, setErrors] = useState<{ name?: string; email?: string; otp?: string; form?: string }>(
    {},
  );
  const [done, setDone] = useState<{ token?: string } | null>(null);

  const pending =
    guestMut.isPending || userMut.isPending || requestOtpMut.isPending || verifyOtpMut.isPending;
  const isFull = event.registration_count >= event.capacity;

  useEffect(() => {
    if (cooldownLeft <= 0) return;
    const id = window.setInterval(() => {
      setCooldownLeft((s) => (s <= 1 ? 0 : s - 1));
    }, 1000);
    return () => window.clearInterval(id);
  }, [cooldownLeft]);

  function validateSignedIn(): boolean {
    setErrors({});
    return true;
  }

  async function onSignedInSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (pending) return;
    if (!validateSignedIn()) return;

    try {
      await userMut.mutateAsync({
        eventId: event.id,
        values: {
          name: name || undefined,
        },
      });
      setDone({});
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setErrors({ form: err.message });
      }
    }
  }

  function validateGuestDetails(): boolean {
    const parsed = RequestGuestOTPInput.safeParse({ name, email });
    if (!parsed.success) {
      const fieldErrors: typeof errors = {};
      for (const issue of parsed.error.issues) {
        const key = issue.path[0];
        if (key === "name" || key === "email") fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return false;
    }
    setErrors({});
    return true;
  }

  async function onRequestCode(e: React.FormEvent) {
    e.preventDefault();
    if (pending) return;
    if (!validateGuestDetails()) return;

    try {
      await requestOtpMut.mutateAsync({
        eventId: event.id,
        values: { name: name.trim(), email: email.trim() },
      });
      setGuestStep("code");
      setOtp("");
      setCooldownLeft(RESEND_COOLDOWN_SECONDS);
      setErrors({});
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 429) {
          setCooldownLeft(RESEND_COOLDOWN_SECONDS);
          setErrors({ email: err.message || "Please wait before requesting another code" });
          return;
        }
        if (err.status === 409) {
          setErrors({ email: err.message });
          return;
        }
        if (err.status === 503) {
          setErrors({ form: "Registration is temporarily unavailable. Please try again shortly." });
          return;
        }
        setErrors({ form: err.message });
      }
    }
  }

  async function onResendCode() {
    if (pending || cooldownLeft > 0) return;
    if (!validateGuestDetails()) return;

    try {
      await requestOtpMut.mutateAsync({
        eventId: event.id,
        values: { name: name.trim(), email: email.trim() },
      });
      setCooldownLeft(RESEND_COOLDOWN_SECONDS);
      setErrors({});
    } catch (err) {
      if (err instanceof ApiError && err.status === 429) {
        setCooldownLeft(RESEND_COOLDOWN_SECONDS);
        setErrors({ otp: err.message || "Please wait before requesting another code" });
        return;
      }
      if (err instanceof ApiError) {
        setErrors({ otp: err.message });
      }
    }
  }

  async function onVerifyAndRegister(e: React.FormEvent) {
    e.preventDefault();
    if (pending) return;

    const parsedOtp = VerifyGuestOTPInput.safeParse({ email: email.trim(), otp: otp.trim() });
    if (!parsedOtp.success) {
      setErrors({ otp: parsedOtp.error.issues[0]?.message ?? "Enter the 6-digit code" });
      return;
    }

    const guestParsed = CreateGuestRegistrationInput.safeParse({
      name: name.trim(),
      email: email.trim(),
    });
    if (!guestParsed.success) {
      setGuestStep("details");
      setErrors({ name: guestParsed.error.issues[0]?.message });
      return;
    }

    try {
      await verifyOtpMut.mutateAsync({
        eventId: event.id,
        values: parsedOtp.data,
      });
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 400) {
          setErrors({ otp: err.message || "Invalid or expired code" });
          return;
        }
        if (err.status === 503) {
          setErrors({ form: "Registration is temporarily unavailable. Please try again shortly." });
          return;
        }
        setErrors({ form: err.message });
      }
      return;
    }

    try {
      const reg = await guestMut.mutateAsync({
        eventId: event.id,
        values: guestParsed.data,
      });
      setDone({ token: reg.access_token });
    } catch (err) {
      if (err instanceof ApiError) {
        if (
          err.status === 403 ||
          (err.status === 400 && /not verified|verified/i.test(err.message))
        ) {
          setErrors({
            otp: "Verification expired. Request a new code and try again.",
          });
          return;
        }
        if (err.status === 409) {
          setErrors({ form: err.message });
          return;
        }
        if (err.status === 503) {
          setErrors({ form: "Registration is temporarily unavailable. Please try again shortly." });
          return;
        }
        setErrors({ form: err.message });
      }
    }
  }

  function backToDetails() {
    setGuestStep("details");
    setOtp("");
    setErrors({});
  }

  if (isFull) {
    return (
      <div className="rounded-xl bg-muted p-6 text-center">
        <p className="display-headline text-[1.0625rem]">This event is full</p>
        <p className="mt-1 text-sm text-muted-foreground">
          All seats are currently reserved. Check back later for cancellations.
        </p>
      </div>
    );
  }

  if (done) {
    return (
      <div className="flex flex-col items-center gap-3 rounded-2xl border bg-emerald-50/60 p-6 text-center">
        <CheckCircle2 className="size-8 text-emerald-600" />
        <div>
          <p className="display-headline text-[1.0625rem] text-foreground">You're registered!</p>
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

  if (isSignedIn) {
    return (
      <form onSubmit={onSignedInSubmit} className="flex flex-col gap-5">
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="reg-email">Email</FieldLabel>
            <Input
              id="reg-email"
              type="email"
              value={defaultEmail}
              readOnly
              disabled
              autoComplete="email"
            />
            <FieldDescription>
              Tickets and updates go to your verified account email.
            </FieldDescription>
          </Field>
        </FieldGroup>

        <Button type="submit" size="lg" className="w-full" disabled={pending}>
          {pending && <Spinner data-icon="inline-start" />}
          {event.requires_approval ? "Request to register" : "Get tickets"}
        </Button>
        {errors.form && <p className="text-sm text-destructive">{errors.form}</p>}
        <p className="text-center text-xs text-muted-foreground">
          You'll register with your EnviToo account.
        </p>
      </form>
    );
  }

  if (guestStep === "code") {
    return (
      <form onSubmit={onVerifyAndRegister} className="flex flex-col gap-5">
        <div className="rounded-xl bg-muted/60 px-4 py-3 text-sm">
          <p className="text-muted-foreground">Code sent to</p>
          <p className="font-medium text-foreground">{email.trim()}</p>
          <button
            type="button"
            className="mt-1 text-xs text-muted-foreground underline-offset-2 hover:underline"
            onClick={backToDetails}
            disabled={pending}
          >
            Change email
          </button>
        </div>

        <FieldGroup>
          <Field data-invalid={!!errors.otp}>
            <FieldLabel htmlFor="reg-otp">Verification code</FieldLabel>
            <Input
              id="reg-otp"
              value={otp}
              onChange={(e) => setOtp(e.target.value.replace(/\D/g, "").slice(0, 6))}
              placeholder="6-digit code"
              inputMode="numeric"
              autoComplete="one-time-code"
              aria-invalid={!!errors.otp}
              maxLength={6}
            />
            {errors.otp ? (
              <FieldError>{errors.otp}</FieldError>
            ) : (
              <FieldDescription>Enter the 6-digit code from your email.</FieldDescription>
            )}
          </Field>
        </FieldGroup>

        {errors.form && <p className="text-sm text-destructive">{errors.form}</p>}

        <Button type="submit" size="lg" className="w-full" disabled={pending || otp.trim().length !== 6}>
          {pending && <Spinner data-icon="inline-start" />}
          {event.requires_approval ? "Verify & request" : "Verify & get tickets"}
        </Button>

        <Button
          type="button"
          variant="ghost"
          className="w-full"
          disabled={pending || cooldownLeft > 0}
          onClick={onResendCode}
        >
          {cooldownLeft > 0 ? `Resend code in ${cooldownLeft}s` : "Resend code"}
        </Button>
      </form>
    );
  }

  return (
    <form onSubmit={onRequestCode} className="flex flex-col gap-5">
      <FieldGroup>
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
            <FieldDescription>We'll email a 6-digit code to confirm it's you.</FieldDescription>
          )}
        </Field>
      </FieldGroup>

      {errors.form && <p className="text-sm text-destructive">{errors.form}</p>}

      <Button type="submit" size="lg" className="w-full" disabled={pending}>
        {pending && <Spinner data-icon="inline-start" />}
        Send verification code
      </Button>
      <p className="text-center text-xs text-muted-foreground">
        No account needed — register as a guest.
      </p>
    </form>
  );
}
