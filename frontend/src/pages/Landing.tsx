import { useMemo, useRef } from "react";
import { Link } from "react-router-dom";
import { motion, useReducedMotion, useScroll, useTransform } from "framer-motion";
import {
  ArrowUpRight,
  BookOpen,
  Briefcase,
  CalendarPlus,
  Compass,
  Dumbbell,
  Globe,
  Heart,
  Monitor,
  Music,
  Palette,
  QrCode,
  Share2,
  Ticket,
  Users,
  UtensilsCrossed,
  Zap,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS } from "@/lib/api";
import { cn } from "@/lib/utils";

export function Landing() {
  const reduceMotion = useReducedMotion();
  const transition = reduceMotion
    ? { duration: 0 }
    : { duration: 0.45, ease: [0.22, 1, 0.36, 1] as const };

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
  const bgY = useTransform(scrollYProgress, [0, 1], ["0%", "25%"]);
  const reduceMotion = useReducedMotion();

  return (
    <section
      ref={heroRef}
      className="relative min-h-dvh overflow-hidden border-b border-border/30 pt-16"
    >
      <motion.div
        className="pointer-events-none absolute inset-0 bg-light-teal"
        style={reduceMotion ? {} : { y: bgY }}
        aria-hidden
      />
      <div className="pointer-events-none absolute inset-0 bg-grain opacity-[0.025]" aria-hidden />

      <PageContainer className="relative flex min-h-[calc(100dvh-4rem)] items-center py-20 sm:py-28">
        <div className="flex w-full flex-col items-start gap-12 lg:flex-row lg:items-center lg:justify-between">
          <div className="max-w-xl">
            <h1 className="animate-fade-up display-hero stagger-1 text-[clamp(2.5rem,5vw,4.5rem)] leading-[0.95] tracking-[-0.035em]">
              Host events people
              <br />
              <span className="text-primary">actually</span> show up for.
            </h1>
            <p className="animate-fade-up stagger-2 mt-6 max-w-[32rem] text-lg leading-relaxed tracking-[-0.014em] text-muted-foreground">
              Better discovery, cleaner tickets, no subscriptions. Publish a page
              in minutes and open registrations today.
            </p>
            <div className="animate-fade-up stagger-3 mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
              <Button asChild size="lg">
                <Link to="/create">
                  Create your first event
                  <ArrowUpRight className="size-4" />
                </Link>
              </Button>
            </div>
          </div>

          <motion.div
            className="hidden w-full max-w-sm shrink-0 lg:block"
            initial={{ opacity: 0, y: 24 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1], delay: 0.3 }}
          >
            <EventCardMockup />
          </motion.div>
        </div>
      </PageContainer>

      <div className="absolute bottom-6 left-1/2 hidden -translate-x-1/2 sm:block">
        <div className="flex flex-col items-center gap-2 text-xs text-muted-foreground/50">
          <span className="block size-1 rounded-full bg-muted-foreground/30" />
          <span className="block size-1 rounded-full bg-muted-foreground/20" />
          <span className="block size-1 rounded-full bg-muted-foreground/10" />
        </div>
      </div>
    </section>
  );
}

function EventCardMockup() {
  const reduceMotion = useReducedMotion();
  return (
    <motion.div
      className="relative"
      animate={
        reduceMotion ? {} : { y: [0, -6, 0] }
      }
      transition={{
        duration: 5,
        ease: "easeInOut",
        repeat: Infinity,
      }}
    >
      <div className="absolute -inset-8 rounded-[2rem] bg-primary/5 blur-2xl" aria-hidden />

      <div className="relative overflow-hidden rounded-2xl border border-border/50 bg-card shadow-2xl">
        <div className="bg-gradient-to-br from-primary/30 via-primary/10 to-card p-5 pb-8">
          <div className="flex items-center justify-between">
            <span className="inline-flex items-center gap-1.5 rounded-md bg-black/20 px-2 py-1 text-[11px] font-medium tracking-wide text-white/80 backdrop-blur">
              <Ticket className="size-3" />
              Ticket
            </span>
            <span className="rounded-md bg-emerald-400/20 px-2 py-1 text-[11px] font-medium text-emerald-300">
              Free
            </span>
          </div>
          <h3 className="mt-4 font-serif text-xl font-semibold leading-tight text-white">
            Summer Design Meetup
          </h3>
          <p className="mt-1.5 text-sm text-white/70">August 15, 2026 · 6:00 PM</p>
        </div>

        <div className="flex items-center gap-4 p-5">
          <div className="grid size-16 shrink-0 place-items-center rounded-xl bg-white p-1.5">
            <svg viewBox="0 0 100 100" className="size-full">
              <rect x="10" y="10" width="80" height="80" rx="8" fill="#14b8a6" opacity="0.15" />
              <rect x="20" y="20" width="60" height="60" rx="4" fill="#14b8a6" opacity="0.3" />
              <rect x="30" y="30" width="40" height="40" rx="2" fill="#14b8a6" opacity="0.5" />
            </svg>
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">Alex Rivera</p>
            <p className="text-xs text-muted-foreground">Registered · Seat 42 of 100</p>
          </div>
        </div>

        <div className="relative flex items-center px-5">
          <div className="absolute -left-2.5 top-1/2 size-5 -translate-y-1/2 rounded-full bg-background" />
          <div className="h-px flex-1 border-t border-dashed border-border/60" />
          <div className="absolute -right-2.5 top-1/2 size-5 -translate-y-1/2 rounded-full bg-background" />
        </div>

        <div className="flex items-center justify-between p-5 pt-4">
          <div className="flex items-center gap-3">
            <div className="grid size-10 place-items-center rounded-lg border border-border/50 bg-muted/30">
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

const CATEGORY_ICONS: Record<string, React.ComponentType<{ className?: string; strokeWidth?: number }>> = {
  music: Music,
  tech: Monitor,
  food: UtensilsCrossed,
  sports: Dumbbell,
  arts: Palette,
  education: BookOpen,
  social: Users,
  charity: Heart,
  business: Briefcase,
  other: Globe,
};

function CategorySection() {
  return (
    <section className="border-b border-border/30 py-16 sm:py-20" aria-label="Event categories">
      <PageContainer>
        <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between sm:gap-0">
          <div>
            <p className="text-xs tracking-wide text-muted-foreground/60 uppercase">Categories</p>
            <h2 className="display-title mt-1 text-3xl sm:text-4xl">
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
                "animate-fade-up group relative flex items-center gap-3 rounded-xl border border-self-weak bg-card/50 p-4 transition-all hover:border-primary/30 hover:bg-card",
                `stagger-${Math.min(i + 1, 6)}`,
              )}
            >
              <span className="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary" aria-hidden>
                {(() => {
                  const Icon = CATEGORY_ICONS[cat] ?? Globe;
                  return <Icon className="size-4" strokeWidth={1.5} />;
                })()}
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
    icon: CalendarPlus,
    title: "Create & publish",
    body: "Title, schedule, capacity, cover — ship a polished event page without the busywork.",
    className: "sm:col-span-2 sm:row-span-1",
    large: true,
  },
  {
    icon: Ticket,
    title: "QR check-in",
    body: "Guests get QR tickets. You scan at the door. Approvals and cancellations stay simple.",
  },
  {
    icon: Users,
    title: "Volunteer roster",
    body: "Open roles, review applications, and keep staff aligned from one dashboard.",
  },
  {
    icon: Compass,
    title: "Discover or unlisted",
    body: "Feature public events on Discover, or share a private link for invite-only nights.",
  },
];

function FeaturesSection({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  return (
    <section className="relative border-b border-border/30 py-20 sm:py-28">
      <div className="pointer-events-none absolute inset-0 bg-light-amber" aria-hidden />

      <PageContainer className="relative">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="display-title max-w-2xl text-3xl sm:text-4xl">
              For hosts, guests, <span className="italic text-primary">and</span> volunteers
            </h2>
            <p className="mt-3 max-w-lg text-sm leading-relaxed text-muted-foreground">
              One platform that handles creation, registration, check-in, and coordination.
            </p>
          </div>
        </div>

        <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f, i) => (
            <motion.div
              key={f.title}
              className={cn(
                "group rounded-xl border border-self-weak bg-card/60 p-6 transition-colors hover:border-primary/20 hover:bg-card",
                f.className,
                f.large && "flex flex-col justify-between sm:min-h-[16rem]",
              )}
              initial="hidden"
              whileInView="visible"
              viewport={{ once: true, margin: "-60px" }}
              variants={{
                hidden: { opacity: 0, y: 12 },
                visible: { opacity: 1, y: 0 },
              }}
              transition={{ ...transition, delay: i * 0.06 }}
            >
              <f.icon
                className={cn(
                  "text-primary/60",
                  f.large ? "mb-5 size-6" : "mb-4 size-5",
                )}
                strokeWidth={1.5}
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
                  <h3 className="display-headline-sans text-base font-semibold">{f.title}</h3>
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
    icon: Zap,
    title: "Create your page",
    body: "Add details, set capacity, choose public or unlisted. Publish when you're ready.",
  },
  {
    icon: Share2,
    title: "Share the link",
    body: "Send one URL. Guests register in seconds — signed-in or as a guest.",
  },
  {
    icon: QrCode,
    title: "Manage the day",
    body: "Approve registrations, scan QR tickets, and coordinate volunteers from the dashboard.",
  },
];

function HowItWorksSection() {
  return (
    <section className="border-b border-border/30 py-20 sm:py-28">
      <PageContainer>
        <div className="mx-auto max-w-2xl">
          <p className="text-xs tracking-wide text-muted-foreground/60 uppercase">Process</p>
          <h2 className="display-title mt-1 text-3xl sm:text-4xl">
            Live in three steps
          </h2>
          <p className="mt-4 max-w-md text-base leading-relaxed text-muted-foreground">
            From blank page to door check-in without the busywork.
          </p>
        </div>

        <div className="mt-16 grid gap-6 sm:grid-cols-3 sm:gap-8">
          {STEPS.map((step, i) => (
            <div
              key={step.title}
              className={cn(
                "animate-fade-up relative flex flex-col items-start text-left",
                `stagger-${Math.min(i + 1, 6)}`,
              )}
            >
              <div className="flex size-14 items-center justify-center rounded-2xl border border-border/40 bg-card">
                <step.icon className="size-6 text-primary/70" strokeWidth={1.5} />
              </div>
              <div className="mt-2 flex items-center gap-2">
                {i > 0 && (
                  <span
                    className="hidden h-px w-8 bg-border/40 sm:block"
                    aria-hidden
                  />
                )}
              </div>
              <p className="mt-5 font-serif text-lg font-semibold">{step.title}</p>
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
    <section className="relative border-b border-border/30 py-20 sm:py-28">
      <div className="pointer-events-none absolute inset-0 bg-light-cyan" aria-hidden />
      <PageContainer className="relative">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-xs tracking-wide text-muted-foreground/60 uppercase">Events</p>
            <h2 className="display-title mt-1 text-3xl sm:text-4xl">Featured events</h2>
          </div>
          <Link
            to="/discover"
            className="hidden items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground sm:inline-flex"
          >
            View all
            <ArrowUpRight className="size-3.5" />
          </Link>
        </div>

        <div className="mt-10 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
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
    <section className="relative py-20 sm:py-28">
      <div className="pointer-events-none absolute inset-0 bg-light-rose" aria-hidden />
      <PageContainer className="relative">
        <div className="mx-auto max-w-2xl rounded-2xl border border-border/30 bg-card/50 p-8 text-center sm:p-12">
          <h2 className="display-title text-2xl sm:text-3xl">
            Ready to host your first event?
          </h2>
          <p className="mt-4 max-w-md mx-auto text-base leading-relaxed text-muted-foreground">
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
