import { useMemo, useRef } from "react";
import { Link } from "react-router-dom";
import { motion, useReducedMotion, useScroll, useTransform } from "framer-motion";
import {
  ArrowUpRight,
  Ticket,
} from "lucide-react";
import {
  CalendarCheckIcon,
  CompassIcon,
  LinkIcon,
  ScanTextIcon,
  UsersIcon,
  ZapIcon,
} from "lucide-animated";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { CategoryIcon } from "@/components/domain/CategoryIcon";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS } from "@/lib/api";
import { cn } from "@/lib/utils";

export function Landing() {
  const reduceMotion = useReducedMotion();
  const transition = reduceMotion
    ? { duration: 0 }
    : { duration: 0.4, ease: [0.22, 1, 0.36, 1] as const };

  return (
    <div className="overflow-x-hidden">
      <HeroSection />
      <CategorySection />
      <FeaturesSection transition={transition} />
      <HowItWorksSection />
      <FeaturedEventsSection />
      <CtaSection />
    </div>
  );
}

// ── Hero ──────────────────────────────────────────────

function HeroSection() {
  const heroRef = useRef<HTMLDivElement>(null);
  const { scrollYProgress } = useScroll({
    target: heroRef,
    offset: ["start start", "end start"],
  });
  const bgY = useTransform(scrollYProgress, [0, 1], ["0%", "18%"]);
  const reduceMotion = useReducedMotion();

  return (
    <section
      ref={heroRef}
      className="relative min-h-dvh overflow-hidden border-b border-border pt-14"
    >
      <motion.div
        className="pointer-events-none absolute inset-0 bg-light-teal"
        style={reduceMotion ? {} : { y: bgY }}
        aria-hidden
      />

      <PageContainer className="relative flex min-h-[calc(100dvh-4rem)] items-center py-20 sm:py-28">
        <div className="flex w-full flex-col items-start gap-12 lg:flex-row lg:items-center lg:justify-between">
          <div className="max-w-xl">
            <p className="animate-fade-up text-eyebrow stagger-1">EnviToo</p>
            <h1 className="animate-fade-up display-hero stagger-2 mt-4 text-[clamp(2.5rem,5vw,4.25rem)]">
              Host events people
              <br />
              <span className="underline decoration-muted-foreground/50 underline-offset-[0.15em]">actually</span> show up for.
            </h1>
            <p className="animate-fade-up text-lede stagger-3 mt-6 max-w-[32rem]">
              Better discovery, cleaner tickets, no subscriptions. Publish a page
              in minutes and open registrations today.
            </p>
            <div className="animate-fade-up stagger-4 mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
              <Button asChild size="lg">
                <Link to="/create">
                  Create your first event
                  <ArrowUpRight className="size-4" />
                </Link>
              </Button>
              <Button asChild variant="outline" size="lg">
                <Link to="/discover">Browse events</Link>
              </Button>
            </div>
          </div>

          <motion.div
            className="hidden w-full max-w-sm shrink-0 lg:block"
            initial={reduceMotion ? false : { opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: reduceMotion ? 0 : 0.5, ease: [0.22, 1, 0.36, 1], delay: 0.2 }}
          >
            <EventCardMockup />
          </motion.div>
        </div>
      </PageContainer>
    </section>
  );
}

function EventCardMockup() {
  const reduceMotion = useReducedMotion();
  return (
    <motion.div
      className="relative"
      animate={reduceMotion ? {} : { y: [0, -5, 0] }}
      transition={{
        duration: 5.5,
        ease: "easeInOut",
        repeat: Infinity,
      }}
    >
      <div className="relative overflow-hidden rounded-xl border border-border bg-card shadow-sm">
        <div className="bg-gradient-to-br from-primary/25 via-primary/8 to-card p-5 pb-8">
          <div className="flex items-center justify-between">
            <span className="inline-flex items-center gap-1.5 rounded-md bg-black/15 px-2 py-1 text-[11px] font-medium tracking-wide text-white/85 backdrop-blur-sm">
              <Ticket className="size-3" />
              Ticket
            </span>
            <span className="rounded-md bg-emerald/15 px-2 py-1 text-[11px] font-medium text-emerald">
              Free
            </span>
          </div>
          <h3 className="display-headline mt-4 text-xl text-white">
            Summer Design Meetup
          </h3>
          <p className="mt-1.5 text-sm text-white/70">August 15, 2026 · 6:00 PM</p>
        </div>

        <div className="flex items-center gap-4 p-5">
          <div className="grid size-16 shrink-0 place-items-center rounded-lg bg-muted p-1.5">
            <svg viewBox="0 0 100 100" className="size-full text-primary">
              <rect x="10" y="10" width="80" height="80" rx="8" fill="currentColor" opacity="0.12" />
              <rect x="20" y="20" width="60" height="60" rx="4" fill="currentColor" opacity="0.25" />
              <rect x="30" y="30" width="40" height="40" rx="2" fill="currentColor" opacity="0.45" />
            </svg>
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">Alex Rivera</p>
            <p className="text-xs text-muted-foreground">Registered · Seat 42 of 100</p>
          </div>
        </div>

        <div className="relative flex items-center px-5">
          <div className="absolute -left-2.5 top-1/2 size-5 -translate-y-1/2 rounded-full bg-background" />
          <div className="h-px flex-1 border-t border-dashed border-border" />
          <div className="absolute -right-2.5 top-1/2 size-5 -translate-y-1/2 rounded-full bg-background" />
        </div>

        <div className="flex items-center justify-between p-5 pt-4">
          <div className="flex items-center gap-3">
            <div className="grid size-10 place-items-center rounded-md border border-border bg-muted/40">
              <svg viewBox="0 0 24 24" className="size-5 text-muted-foreground/60">
                <rect x="3" y="3" width="7" height="7" rx="1" fill="currentColor" opacity="0.3" />
                <rect x="14" y="3" width="7" height="7" rx="1" fill="currentColor" opacity="0.3" />
                <rect x="3" y="14" width="7" height="7" rx="1" fill="currentColor" opacity="0.3" />
                <rect x="14" y="14" width="3" height="3" rx="0.5" fill="currentColor" opacity="0.5" />
                <rect x="19" y="14" width="2" height="3" rx="0.5" fill="currentColor" opacity="0.5" />
                <rect x="14" y="19" width="3" height="2" rx="0.5" fill="currentColor" opacity="0.5" />
                <rect x="19" y="19" width="2" height="2" rx="0.5" fill="currentColor" opacity="0.5" />
              </svg>
            </div>
            <div className="text-xs text-muted-foreground">
              <p className="font-medium text-foreground">envitoo.app/e/meetup-2026</p>
              <p>Show at check-in</p>
            </div>
          </div>
          <span className="text-[10px] text-muted-foreground">ID: b7f3a2c1</span>
        </div>
      </div>
    </motion.div>
  );
}

// ── Categories ────────────────────────────────────────

function CategorySection() {
  const reduceMotion = useReducedMotion();

  return (
    <section className="border-b border-border py-16 sm:py-20" aria-label="Event categories">
      <PageContainer>
        <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between sm:gap-0">
          <div>
            <p className="text-eyebrow">Categories</p>
            <h2 className="display-title mt-2 text-3xl sm:text-4xl">
              Browse by category
            </h2>
          </div>
          <Link
            to="/discover"
            className="hidden items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground sm:inline-flex"
          >
            View all
            <ArrowUpRight className="size-3.5" />
          </Link>
        </div>

        <div className="mt-10 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          {CATEGORIES.map((cat, i) => (
            <Link
              key={cat}
              to={`/discover?category=${cat}`}
              className={cn(
                "animate-fade-up group flex items-center gap-3 rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/35 hover:bg-muted/40",
                `stagger-${Math.min(i + 1, 6)}`,
              )}
            >
              <span className="flex size-8 items-center justify-center rounded-md bg-primary/10 text-primary" aria-hidden>
                <CategoryIcon category={cat} animateOnHover={!reduceMotion} />
              </span>
              <span className="text-sm font-medium">{CATEGORY_LABELS[cat]}</span>
            </Link>
          ))}
        </div>

        <Link
          to="/discover"
          className="mt-6 inline-flex items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground sm:hidden"
        >
          View all categories
          <ArrowUpRight className="size-3.5" />
        </Link>
      </PageContainer>
    </section>
  );
}

// ── Features (bento layout) ───────────────────────────

const FEATURES = [
  {
    Icon: CalendarCheckIcon,
    title: "Create & publish",
    body: "Title, schedule, capacity, cover — ship a polished event page without the busywork.",
    className: "sm:col-span-2 sm:row-span-1",
    large: true,
    size: 28,
  },
  {
    Icon: ScanTextIcon,
    title: "QR check-in",
    body: "Guests get QR tickets. You scan at the door. Approvals and cancellations stay simple.",
    className: "",
    large: false,
    size: 24,
  },
  {
    Icon: UsersIcon,
    title: "Volunteer roster",
    body: "Open roles, review applications, and keep staff aligned from one dashboard.",
    className: "",
    large: false,
    size: 24,
  },
  {
    Icon: CompassIcon,
    title: "Discover or unlisted",
    body: "Feature public events on Discover, or share a private link for invite-only nights.",
    className: "",
    large: false,
    size: 24,
  },
] as const;

function FeaturesSection({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  const reduceMotion = useReducedMotion();

  return (
    <section className="relative border-b border-border py-20 sm:py-28">
      <PageContainer className="relative">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="display-title max-w-2xl text-3xl sm:text-4xl">
              For hosts, guests, and volunteers
            </h2>
            <p className="mt-3 max-w-lg text-sm leading-relaxed text-muted-foreground">
              One platform that handles creation, registration, check-in, and coordination.
            </p>
          </div>
        </div>

        <div className="mt-12 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f, i) => (
            <motion.div
              key={f.title}
              className={cn(
                "group rounded-lg border border-border bg-card p-6 transition-colors hover:border-primary/30",
                f.className,
                f.large && "flex flex-col justify-between sm:min-h-[15rem]",
              )}
              initial={reduceMotion ? false : "hidden"}
              whileInView="visible"
              viewport={{ once: true, margin: "-60px" }}
              variants={{
                hidden: { opacity: 0, y: 10 },
                visible: { opacity: 1, y: 0 },
              }}
              transition={{ ...transition, delay: i * 0.05 }}
            >
              <f.Icon
                size={f.size}
                className={cn("text-primary", f.large ? "mb-5" : "mb-4")}
                animateOnHover={!reduceMotion}
              />
              {f.large ? (
                <>
                  <h3 className="display-headline text-2xl">{f.title}</h3>
                  <p className="mt-3 max-w-md text-sm leading-relaxed text-muted-foreground">
                    {f.body}
                  </p>
                </>
              ) : (
                <>
                  <h3 className="display-headline-sans text-base">{f.title}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {f.body}
                  </p>
                </>
              )}
            </motion.div>
          ))}
        </div>
      </PageContainer>
    </section>
  );
}

// ── How It Works ──────────────────────────────────────

const STEPS = [
  {
    Icon: ZapIcon,
    title: "Create your page",
    body: "Add details, set capacity, choose public or unlisted. Publish when you're ready.",
  },
  {
    Icon: LinkIcon,
    title: "Share the link",
    body: "Send one URL. Guests register in seconds — signed-in or as a guest.",
  },
  {
    Icon: ScanTextIcon,
    title: "Manage the day",
    body: "Approve registrations, scan QR tickets, and coordinate volunteers from the dashboard.",
  },
] as const;

function HowItWorksSection() {
  const reduceMotion = useReducedMotion();

  return (
    <section className="border-b border-border py-20 sm:py-28">
      <PageContainer>
        <div className="max-w-2xl">
          <p className="text-eyebrow">Process</p>
          <h2 className="display-title mt-2 text-3xl sm:text-4xl">
            Live in three steps
          </h2>
          <p className="mt-4 max-w-md text-base leading-relaxed text-muted-foreground">
            From blank page to door check-in without the busywork.
          </p>
        </div>

        <div className="mt-16 grid gap-8 sm:grid-cols-3 sm:gap-10">
          {STEPS.map((step, i) => (
            <div
              key={step.title}
              className={cn(
                "animate-fade-up relative flex flex-col items-start text-left",
                `stagger-${Math.min(i + 1, 6)}`,
              )}
            >
              <div className="flex size-12 items-center justify-center rounded-lg border border-border bg-card text-primary">
                <step.Icon size={24} animateOnHover={!reduceMotion} />
              </div>
              <p className="display-headline mt-5 text-lg">{step.title}</p>
              <p className="mt-2 max-w-xs text-sm leading-relaxed text-muted-foreground">
                {step.body}
              </p>
            </div>
          ))}
        </div>

        <div className="mt-12">
          <Button asChild>
            <Link to="/create">
              Start free
              <ArrowUpRight className="size-4" />
            </Link>
          </Button>
        </div>
      </PageContainer>
    </section>
  );
}

// ── Featured Events ───────────────────────────────────

function FeaturedEventsSection() {
  const nowIso = useMemo(() => new Date().toISOString(), []);
  const { data, isLoading, isError } = usePublicEvents({
    startsAfter: nowIso,
    limit: 6,
  });

  const events = useMemo(
    () =>
      (data?.events ?? [])
        .filter((e) => e.status === "published" && e.registration_count < e.capacity)
        .slice(0, 6),
    [data?.events],
  );

  return (
    <section className="border-b border-border py-20 sm:py-28">
      <PageContainer>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-eyebrow">Events</p>
            <h2 className="display-title mt-2 text-3xl sm:text-4xl">Featured events</h2>
          </div>
          <Link
            to="/discover"
            className="hidden items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground sm:inline-flex"
          >
            View all
            <ArrowUpRight className="size-3.5" />
          </Link>
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {isLoading
            ? Array.from({ length: 3 }).map((_, i) => <EventCardSkeleton key={i} />)
            : isError
              ? null
              : events.map((event, i) => (
                  <EventCard key={event.id} event={event} index={i} />
                ))}
        </div>
      </PageContainer>
    </section>
  );
}

// ── Bottom CTA ────────────────────────────────────────

function CtaSection() {
  return (
    <section className="py-20 sm:py-28">
      <PageContainer>
        <div className="mx-auto max-w-2xl border border-border bg-card p-8 text-center sm:p-12 rounded-xl">
          <h2 className="display-title text-2xl sm:text-3xl">
            Ready to host your first event?
          </h2>
          <p className="mx-auto mt-4 max-w-md text-base leading-relaxed text-muted-foreground">
            Set up your first event page now. Free to start — no subscriptions, no credit card.
          </p>
          <div className="mt-8">
            <Button asChild size="lg">
              <Link to="/create">
                Create your first event
                <ArrowUpRight className="size-4" />
              </Link>
            </Button>
          </div>
        </div>
      </PageContainer>
    </section>
  );
}
