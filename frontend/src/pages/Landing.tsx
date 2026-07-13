import { useMemo } from "react";
import { Link } from "react-router-dom";
import { motion, useReducedMotion } from "framer-motion";
import {
  ArrowRight,
  CalendarPlus,
  QrCode,
  Share2,
  Users,
  Compass,
  Ticket,
  Shield,
  Zap,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS } from "@/lib/api";
import { cn } from "@/lib/utils";

const fadeUp = {
  hidden: { opacity: 0, y: 16 },
  visible: { opacity: 1, y: 0 },
};

export function Landing() {
  const reduceMotion = useReducedMotion();
  const transition = reduceMotion
    ? { duration: 0 }
    : { duration: 0.45, ease: [0.22, 1, 0.36, 1] as const };

  return (
    <div className="overflow-x-hidden">
      <Hero transition={transition} />
      <StatsStrip />
      <CategoryMarquee />
      <Features transition={transition} />
      <HowItWorks transition={transition} />
      <FeaturedEvents transition={transition} />
      <BottomCta transition={transition} />
    </div>
  );
}

function Hero({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  return (
    <section className="relative border-b border-border/40 bg-radial-fade pt-16">
      <div className="pointer-events-none absolute inset-0 bg-grid opacity-40" aria-hidden />
      <PageContainer className="relative flex min-h-[78dvh] flex-col items-center justify-center py-20 text-center sm:py-28">
        <motion.p
          className="text-ui mb-6 inline-flex items-center gap-2 rounded-full border border-primary/25 bg-primary/10 px-3.5 py-1.5 text-[13px] text-primary"
          initial="hidden"
          animate="visible"
          variants={fadeUp}
          transition={transition}
        >
          <span className="size-1.5 rounded-full bg-cyan-400 shadow-[0_0_8px_#22d3ee]" />
          Event pages · registrations · check-in
        </motion.p>

        <motion.h1
          className="display-hero max-w-[18ch] text-[2.75rem] sm:max-w-3xl sm:text-6xl lg:text-7xl"
          initial="hidden"
          animate="visible"
          variants={fadeUp}
          transition={{ ...transition, delay: reduceDelay(transition, 0.05) }}
        >
          Host events people
          <br />
          <span className="text-gradient">actually show up for.</span>
        </motion.h1>

        <motion.p
          className="text-lede mt-7 max-w-[34rem]"
          initial="hidden"
          animate="visible"
          variants={fadeUp}
          transition={{ ...transition, delay: reduceDelay(transition, 0.1) }}
        >
          Better discovery, cleaner tickets, no subscriptions. Publish a page in
          minutes and open registrations today.
        </motion.p>

        <motion.div
          className="mt-10 flex flex-col items-center gap-3 sm:flex-row"
          initial="hidden"
          animate="visible"
          variants={fadeUp}
          transition={{ ...transition, delay: reduceDelay(transition, 0.15) }}
        >
          <Button asChild size="lg" className="text-ui rounded-full px-8 text-[15px]">
            <Link to="/create">
              Create event
              <ArrowRight data-icon="inline-end" />
            </Link>
          </Button>
          <Button asChild variant="outline" size="lg" className="text-ui rounded-full px-8 text-[15px]">
            <Link to="/discover">Explore events</Link>
          </Button>
        </motion.div>
      </PageContainer>
    </section>
  );
}

const STATS = [
  { value: "7+", label: "Categories", color: "text-teal-300" },
  { value: "QR", label: "Check-in", color: "text-sky-300" },
  { value: "Free", label: "To start", color: "text-emerald-300" },
  { value: "0", label: "Subscriptions", color: "text-amber-300" },
];

function StatsStrip() {
  return (
    <section className="border-b border-border/40">
      <PageContainer className="py-10 sm:py-12">
        <div className="grid grid-cols-2 gap-8 sm:grid-cols-4">
          {STATS.map((s) => (
            <div key={s.label} className="text-center sm:text-left">
              <p className={cn("text-stat text-[2rem] sm:text-[2.5rem]", s.color)}>
                {s.value}
              </p>
              <p className="text-eyebrow mt-2">{s.label}</p>
            </div>
          ))}
        </div>
      </PageContainer>
    </section>
  );
}

const CATEGORY_DOTS: Record<string, string> = {
  technical: "bg-sky-400",
  cultural: "bg-rose-400",
  sports: "bg-emerald-400",
  workshop: "bg-amber-400",
  seminar: "bg-teal-400",
  hackathon: "bg-orange-400",
  other: "bg-zinc-400",
};

function CategoryMarquee() {
  const items = [...CATEGORIES, ...CATEGORIES];

  return (
    <section className="border-b border-border/40 py-8" aria-label="Event categories">
      <div className="relative overflow-hidden">
        <div className="pointer-events-none absolute inset-y-0 left-0 z-10 w-16 bg-gradient-to-r from-background to-transparent sm:w-24" />
        <div className="pointer-events-none absolute inset-y-0 right-0 z-10 w-16 bg-gradient-to-l from-background to-transparent sm:w-24" />
        <div className="marquee-track flex w-max gap-3 px-4">
          {items.map((cat, i) => (
            <Link
              key={`${cat}-${i}`}
              to={`/discover?category=${cat}`}
              className="text-ui inline-flex shrink-0 items-center gap-2 rounded-full border border-border/80 bg-card px-4 py-2 text-[13px] text-muted-foreground transition-colors hover:border-primary/40 hover:bg-primary/5 hover:text-foreground"
            >
              <span
                className={cn(
                  "size-1.5 rounded-full shadow-[0_0_8px_currentColor]",
                  CATEGORY_DOTS[cat] ?? CATEGORY_DOTS.other,
                )}
              />
              {CATEGORY_LABELS[cat]}
            </Link>
          ))}
        </div>
      </div>
    </section>
  );
}

const FEATURES = [
  {
    icon: CalendarPlus,
    title: "Create & publish in minutes",
    body: "Title, schedule, capacity, cover — ship a polished event page without the busywork.",
    className: "sm:col-span-2",
    iconClass: "border-teal-500/30 bg-teal-500/10 text-teal-300",
  },
  {
    icon: Ticket,
    title: "Tickets & QR check-in",
    body: "Guests get QR tickets. You scan at the door. Approvals and cancellations stay simple.",
    iconClass: "border-sky-500/30 bg-sky-500/10 text-sky-300",
  },
  {
    icon: Users,
    title: "Volunteers & roster",
    body: "Open roles, review applications, and keep staff aligned from one dashboard.",
    iconClass: "border-amber-500/30 bg-amber-500/10 text-amber-300",
  },
  {
    icon: Compass,
    title: "Discover or stay unlisted",
    body: "Feature public events on Discover, or share a private link for invite-only nights.",
    className: "sm:col-span-2",
    iconClass: "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
  },
];

function Features({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  return (
    <section className="relative border-b border-border/40 py-20 sm:py-24">
      <div className="pointer-events-none absolute inset-0 bg-mesh opacity-60" aria-hidden />
      <PageContainer className="relative">
        <motion.div
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-80px" }}
          variants={fadeUp}
          transition={transition}
        >
          <p className="text-eyebrow text-primary/90">Everything you need</p>
          <h2 className="display-title mt-3 max-w-xl text-3xl sm:text-4xl">
            One place for hosts, guests, and volunteers
          </h2>
          <p className="text-lede mt-4 max-w-lg text-[1.0625rem]">
            From first publish to door check-in — without juggling spreadsheets and group chats.
          </p>
        </motion.div>

        <div className="mt-12 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f, i) => (
            <motion.div
              key={f.title}
              className={cn(
                "group rounded-xl border border-border/80 bg-card/80 p-6 backdrop-blur-sm transition-colors hover:border-primary/30 hover:bg-card",
                f.className,
              )}
              initial="hidden"
              whileInView="visible"
              viewport={{ once: true, margin: "-40px" }}
              variants={fadeUp}
              transition={{ ...transition, delay: reduceDelay(transition, i * 0.05) }}
            >
              <div
                className={cn(
                  "mb-4 grid size-9 place-items-center rounded-lg border",
                  f.iconClass,
                )}
              >
                <f.icon className="size-4" strokeWidth={1.75} />
              </div>
              <h3 className="display-headline text-[1.0625rem]">{f.title}</h3>
              <p className="mt-2 text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
                {f.body}
              </p>
            </motion.div>
          ))}
        </div>
      </PageContainer>
    </section>
  );
}

const STEPS = [
  {
    n: "1",
    title: "Create your page",
    body: "Add details, set capacity, choose public or unlisted. Publish when you're ready.",
    icon: Zap,
    accent: "text-teal-300 border-teal-500/25 bg-teal-500/10",
    stepColor: "text-teal-300",
  },
  {
    n: "2",
    title: "Share the link",
    body: "Send one URL. Guests register in seconds — signed-in or as a guest.",
    icon: Share2,
    accent: "text-sky-300 border-sky-500/25 bg-sky-500/10",
    stepColor: "text-sky-300",
  },
  {
    n: "3",
    title: "Manage the day",
    body: "Approve registrations, scan QR tickets, and coordinate volunteers from the dashboard.",
    icon: QrCode,
    accent: "text-amber-300 border-amber-500/25 bg-amber-500/10",
    stepColor: "text-amber-300",
  },
];

function HowItWorks({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  return (
    <section className="border-b border-border/40 py-20 sm:py-24">
      <PageContainer>
        <motion.div
          className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between"
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-80px" }}
          variants={fadeUp}
          transition={transition}
        >
          <div>
            <p className="text-eyebrow text-sky-300/90">How it works</p>
            <h2 className="display-title mt-3 text-3xl sm:text-4xl">
              Live in three steps
            </h2>
          </div>
          <Link
            to="/create"
            className="text-ui inline-flex items-center gap-1 text-[15px] text-primary transition-colors hover:text-primary/80"
          >
            Start free
            <ArrowRight className="size-4" />
          </Link>
        </motion.div>

        <div className="mt-12 grid gap-4 md:grid-cols-3">
          {STEPS.map((step, i) => (
            <motion.div
              key={step.n}
              className="relative rounded-xl border border-border/80 bg-card p-6 transition-colors hover:border-primary/25"
              initial="hidden"
              whileInView="visible"
              viewport={{ once: true, margin: "-40px" }}
              variants={fadeUp}
              transition={{ ...transition, delay: reduceDelay(transition, i * 0.06) }}
            >
              <div className="flex items-center justify-between">
                <span className={cn("text-ui text-[12px] tabular-nums", step.stepColor)}>
                  Step {step.n}
                </span>
                <span
                  className={cn(
                    "grid size-8 place-items-center rounded-lg border",
                    step.accent,
                  )}
                >
                  <step.icon className="size-4" strokeWidth={1.75} />
                </span>
              </div>
              <h3 className="display-headline mt-6 text-xl">{step.title}</h3>
              <p className="mt-2 text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
                {step.body}
              </p>
            </motion.div>
          ))}
        </div>
      </PageContainer>
    </section>
  );
}

function FeaturedEvents({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  const nowIso = useMemo(() => new Date().toISOString(), []);
  const { data, isLoading } = usePublicEvents({
    startsAfter: nowIso,
    limit: 6,
  });

  const events = useMemo(
    () =>
      (data?.events ?? []).filter(
        (e) => e.status === "published" && e.registration_count < e.capacity,
      ).slice(0, 6),
    [data?.events],
  );

  if (!isLoading && events.length === 0) return null;

  return (
    <section className="border-b border-border/40 py-20 sm:py-24">
      <PageContainer>
        <motion.div
          className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between"
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-80px" }}
          variants={fadeUp}
          transition={transition}
        >
          <div>
            <p className="text-eyebrow text-amber-300/90">Happening soon</p>
            <h2 className="display-title mt-3 text-3xl sm:text-4xl">
              Featured events
            </h2>
          </div>
          <Link
            to="/discover"
            className="text-ui inline-flex items-center gap-1 text-[15px] text-muted-foreground transition-colors hover:text-foreground"
          >
            View all
            <ArrowRight className="size-4" />
          </Link>
        </motion.div>

        <div className="mt-10 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {isLoading
            ? Array.from({ length: 3 }).map((_, i) => <EventCardSkeleton key={i} />)
            : events.map((event, i) => (
                <EventCard key={event.id} event={event} index={i} />
              ))}
        </div>
      </PageContainer>
    </section>
  );
}

function BottomCta({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  return (
    <section className="py-20 sm:py-28">
      <PageContainer>
        <motion.div
          className="border-gradient relative overflow-hidden rounded-2xl bg-card px-6 py-14 text-center sm:px-12"
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-60px" }}
          variants={fadeUp}
          transition={transition}
        >
          <div
            className="pointer-events-none absolute inset-0 bg-radial-fade opacity-90"
            aria-hidden
          />
          <div className="pointer-events-none absolute inset-0 bg-mesh opacity-50" aria-hidden />
          <div className="relative">
            <div className="mx-auto mb-5 grid size-10 place-items-center rounded-full border border-primary/30 bg-primary/10">
              <Shield className="size-4 text-primary" strokeWidth={1.75} />
            </div>
            <h2 className="display-hero mx-auto max-w-lg text-3xl sm:text-5xl">
              Ready to <span className="text-gradient">host?</span>
            </h2>
            <p className="text-lede mx-auto mt-4 max-w-md">
              Set up your first event page now. Free to start — no subscriptions.
            </p>
            <div className="mt-9 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Button asChild size="lg" className="text-ui rounded-full px-8 text-[15px]">
                <Link to="/create">
                  Create your first event
                  <ArrowRight data-icon="inline-end" />
                </Link>
              </Button>
              <Button asChild variant="outline" size="lg" className="text-ui rounded-full px-8 text-[15px]">
                <Link to="/discover">Browse Discover</Link>
              </Button>
            </div>
          </div>
        </motion.div>
      </PageContainer>
    </section>
  );
}

function reduceDelay(
  transition: { duration: number },
  delay: number,
): number {
  return transition.duration === 0 ? 0 : delay;
}
