import { Link } from "react-router-dom";
import { ArrowUpRight, X } from "lucide-react";
import { LinkIcon, ScanTextIcon, ZapIcon } from "lucide-animated";
import { Button } from "@/components/ui/button";
import { useOnboarding } from "@/hooks/use-onboarding";

const STEPS = [
  {
    Icon: ZapIcon,
    title: "Create your event",
    body: "Title, schedule, capacity — publish in minutes. Keep it private or feature it on Discover.",
    href: "/create",
    cta: "Create an event",
  },
  {
    Icon: LinkIcon,
    title: "Share the link",
    body: "Send one URL. Friends register in seconds — no account needed for guests.",
    href: "/discover",
    cta: "Browse events",
  },
  {
    Icon: ScanTextIcon,
    title: "Manage the day",
    body: "Approve registrations, scan QR tickets at the door, and coordinate volunteers.",
    href: "/manage",
    cta: "Open dashboard",
  },
] as const;

export function OnboardingChecklist() {
  const { show, dismiss } = useOnboarding();

  if (!show) return null;

  return (
    <section
      aria-label="Getting started"
      className="relative overflow-hidden rounded-xl border border-primary/25 bg-primary/[0.04] p-5 sm:p-6"
    >
      <Button
        variant="ghost"
        size="icon"
        className="absolute right-3 top-3 text-muted-foreground"
        onClick={dismiss}
        aria-label="Dismiss getting started"
      >
        <X className="size-4" />
      </Button>

      <h2 className="display-headline-sans text-base font-semibold">
        New here? Hosting takes three steps
      </h2>
      <p className="mt-1 max-w-xl text-sm leading-relaxed text-muted-foreground">
        You can dismiss this anytime — your dashboard keeps working.
      </p>

      <div className="mt-5 grid gap-3 sm:grid-cols-3">
        {STEPS.map((step, i) => (
          <Link
            key={step.title}
            to={step.href}
            className="group flex flex-col gap-2 rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/40"
          >
            <div className="flex items-center justify-between">
              <span className="flex size-9 items-center justify-center rounded-md bg-primary/10 text-primary">
                <step.Icon size={20} animateOnHover />
              </span>
              <span className="text-eyebrow text-[11px] text-muted-foreground">
                Step {i + 1}
              </span>
            </div>
            <h3 className="display-headline-sans text-sm font-semibold">{step.title}</h3>
            <p className="text-[13px] leading-relaxed text-muted-foreground">{step.body}</p>
            <span className="mt-auto inline-flex items-center gap-1 pt-1 text-[13px] font-medium text-primary">
              {step.cta}
              <ArrowUpRight className="size-3.5 transition-transform group-hover:translate-x-0.5 group-hover:-translate-y-0.5" />
            </span>
          </Link>
        ))}
      </div>
    </section>
  );
}
