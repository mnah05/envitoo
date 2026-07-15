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
      <CategorySection />
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
    <section className="relative min-h-dvh border-b border-border/40 bg-radial-fade pt-16">
      <div
        className="pointer-events-none absolute inset-0 bg-grain opacity-[0.035]"
        aria-hidden
      />
      <PageContainer className="relative flex min-h-[calc(100dvh-4rem)] flex-col justify-center py-20 sm:py-28">
        <div className="max-w-3xl">
          <motion.h1
            className="display-hero text-[2.75rem] sm:text-6xl lg:text-7xl"
            initial="hidden"
            animate="visible"
            variants={fadeUp}
            transition={transition}
          >
            Host events people
            <br />
            <span className="text-primary">actually</span> show up for.
          </motion.h1>

          <motion.p
            className="text-lede mt-7 max-w-[34rem]"
            initial="hidden"
            animate="visible"
            variants={fadeUp}
            transition={{ ...transition, delay: reduceDelay(transition, 0.08) }}
          >
            Better discovery, cleaner tickets, no subscriptions. Publish a page in
            minutes and open registrations today.
          </motion.p>

          <motion.div
            className="mt-10 flex flex-col items-start gap-4 sm:flex-row sm:items-center"
            initial="hidden"
            animate="visible"
            variants={fadeUp}
            transition={{ ...transition, delay: reduceDelay(transition, 0.14) }}
          >
            <Button asChild size="lg" className="text-ui rounded-full px-8 text-[15px]">
              <Link to="/create">
                Create event
                <ArrowRight data-icon="inline-end" />
              </Link>
            </Button>
            <Link
              to="/discover"
              className="text-ui text-[15px] text-muted-foreground transition-colors hover:text-foreground"
            >
              Explore events
              <ArrowRight className="ml-1 inline size-4" />
            </Link>
          </motion.div>
        </div>
      </PageContainer>
    </section>
  );
}

const STATS = [
  { value: "Free", label: "To start" },
  { value: "QR", label: "Check-in" },
  { value: "0", label: "Subscriptions" },
];

function StatsStrip() {
  return (
    <section className="border-b border-border/40">
      <PageContainer className="py-10 sm:py-12">
        <div className="flex flex-col gap-8 sm:flex-row sm:items-center sm:justify-between sm:gap-12">
          {STATS.map((s, i) => (
            <div
              key={s.label}
              className={cn(
                "flex items-baseline gap-3 sm:flex-col sm:items-start sm:gap-2",
                i > 0 && "sm:border-l sm:border-border/50 sm:pl-12",
              )}
            >
              <p className="text-stat text-[2rem] text-foreground/90 sm:text-[2.5rem]">
                {s.value}
              </p>
              <p className="text-eyebrow">{s.label}</p>
            </div>
          ))}
        </div>
      </PageContainer>
    </section>
  );
}

function CategorySection() {
  return (
    <section className="border-b border-border/40 py-12 sm:py-16" aria-label="Event categories">
      <PageContainer>
        <h2 className="display-title text-2xl sm:text-3xl">Browse by category</h2>
        <div className="mt-8 flex gap-3 overflow-x-auto pb-2 scrollbar-none sm:grid sm:grid-cols-3 sm:overflow-visible lg:grid-cols-4">
          {CATEGORIES.map((cat) => (
            <Link
              key={cat}
              to={`/discover?category=${cat}`}
              className="text-ui flex min-w-[10rem] shrink-0 items-center justify-between rounded-xl border border-border/80 bg-card px-4 py-4 text-[14px] text-muted-foreground transition-colors hover:border-primary/35 hover:bg-primary/5 hover:text-foreground sm:min-w-0"
            >
              {CATEGORY_LABELS[cat]}
              <ArrowRight className="size-3.5 opacity-50" />
            </Link>
          ))}
        </div>
      </PageContainer>
    </section>
  );
}

const FEATURES = [
  {
    icon: CalendarPlus,
    title: "Create & publish in minutes",
    body: "Title, schedule, capacity, cover — ship a polished event page without the busywork.",
    className: "sm:col-span-2 sm:row-span-1",
    large: true,
  },
  {
    icon: Ticket,
    title: "Tickets & QR check-in",
    body: "Guests get QR tickets. You scan at the door. Approvals and cancellations stay simple.",
  },
  {
    icon: Users,
    title: "Volunteers & roster",
    body: "Open roles, review applications, and keep staff aligned from one dashboard.",
  },
  {
    icon: Compass,
    title: "Discover or stay unlisted",
    body: "Feature public events on Discover, or share a private link for invite-only nights.",
  },
];

function Features({
  transition,
}: {
  transition: { duration: number; ease?: readonly [number, number, number, number] };
}) {
  return (
    <section className="relative border-b border-border/40 py-20 sm:py-24">
      <PageContainer className="relative">
        <motion.div
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-80px" }}
          variants={fadeUp}
          transition={transition}
        >
          <h2 className="display-title max-w-xl text-3xl sm:text-4xl">
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
                "group rounded-xl border border-border/80 bg-card/80 p-6 transition-colors hover:border-primary/30 hover:bg-card",
                f.className,
                f.large && "flex flex-col justify-between sm:min-h-[14rem]",
              )}
              initial="hidden"
              whileInView="visible"
              viewport={{ once: true, margin: "-40px" }}
              variants={fadeUp}
              transition={{ ...transition, delay: reduceDelay(transition, i * 0.05) }}
            >
              <f.icon
                className={cn(
                  "text-muted-foreground",
                  f.large ? "mb-6 size-6" : "mb-4 size-5",
                )}
                strokeWidth={1.5}
              />
              {f.large ? (
                <>
                  <h3 className="display-headline text-xl sm:text-2xl">{f.title}</h3>
                  <p className="mt-3 max-w-md text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
                    {f.body}
                  </p>
                </>
              ) : (
                <>
                  <h3 className="display-headline text-[1.0625rem]">{f.title}</h3>
                  <p className="mt-2 text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
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

const STEPS = [
  {
    n: "01",
    title: "Create your page",
    body: "Add details, set capacity, choose public or unlisted. Publish when you're ready.",
    icon: Zap,
  },
  {
    n: "02",
    title: "Share the link",
    body: "Send one URL. Guests register in seconds — signed-in or as a guest.",
    icon: Share2,
  },
  {
    n: "03",
    title: "Manage the day",
    body: "Approve registrations, scan QR tickets, and coordinate volunteers from the dashboard.",
    icon: QrCode,
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
            <h2 className="display-title text-3xl sm:text-4xl">Live in three steps</h2>
            <p className="text-lede mt-3 max-w-md text-[1.0625rem]">
              From blank page to door check-in without the busywork.
            </p>
          </div>
          <Link
            to="/create"
            className="text-ui inline-flex items-center gap-1 text-[15px] text-primary transition-colors hover:text-primary/80"
          >
            Start free
            <ArrowRight className="size-4" />
          </Link>
        </motion.div>

        {/* Horizontal flow with connecting line */}
        <div className="relative mt-14">
          <div
            className="pointer-events-none absolute top-5 right-[16.67%] left-[16.67%] hidden h-px bg-border md:block"
            aria-hidden
          />
          <div className="grid gap-10 md:grid-cols-3 md:gap-6">
            {STEPS.map((step, i) => (
              <motion.div
                key={step.n}
                className="relative flex flex-col items-start"
                initial="hidden"
                whileInView="visible"
                viewport={{ once: true, margin: "-40px" }}
                variants={fadeUp}
                transition={{ ...transition, delay: reduceDelay(transition, i * 0.06) }}
              >
                <div className="relative z-10 flex size-10 items-center justify-center rounded-full border border-border bg-background">
                  <step.icon className="size-4 text-muted-foreground" strokeWidth={1.5} />
                </div>
                <span className="text-ui mt-5 text-[12px] tabular-nums text-muted-foreground">
                  {step.n}
                </span>
                <h3 className="display-headline mt-2 text-xl">{step.title}</h3>
                <p className="mt-2 text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
                  {step.body}
                </p>
              </motion.div>
            ))}
          </div>
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
      (data?.events ?? [])
        .filter((e) => e.status === "published" && e.registration_count < e.capacity)
        .slice(0, 6),
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
            <h2 className="display-title text-3xl sm:text-4xl">Featured events</h2>
            <p className="text-lede mt-3 max-w-md text-[1.0625rem]">
              What&apos;s open next — real events, ready to join.
            </p>
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
    <section className="relative overflow-hidden py-20 sm:py-28">
      <div
        className="pointer-events-none absolute inset-0 bg-grain opacity-[0.03]"
        aria-hidden
      />
      <PageContainer className="relative">
        <motion.div
          className="flex flex-col items-start gap-6 sm:items-center sm:text-center"
          initial="hidden"
          whileInView="visible"
          viewport={{ once: true, margin: "-60px" }}
          variants={fadeUp}
          transition={transition}
        >
          <p className="text-lede max-w-md">
            Set up your first event page now. Free to start — no subscriptions.
          </p>
          <Button asChild size="lg" className="text-ui rounded-full px-8 text-[15px]">
            <Link to="/create">
              Create your first event
              <ArrowRight data-icon="inline-end" />
            </Link>
          </Button>
        </motion.div>

        <p
          className="display-hero pointer-events-none mt-16 select-none text-center text-[clamp(4rem,18vw,12rem)] leading-none tracking-[-0.04em] text-foreground/[0.06]"
          aria-hidden
        >
          EnviToo
        </p>
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
