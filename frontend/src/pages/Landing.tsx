import { Link } from "react-router-dom";
import { motion } from "framer-motion";
import { ArrowRight, CalendarCheck, MapPinned, Sparkles, Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS } from "@/lib/api";

const fadeUp = {
  hidden: { opacity: 0, y: 16 },
  show: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: 0.05 * i, duration: 0.5, ease: [0.22, 1, 0.36, 1] as const },
  }),
};

export function Landing() {
  const nowIso = new Date().toISOString();
  const { data, isLoading, isError, refetch } = usePublicEvents({ startsAfter: nowIso, limit: 8 });
  const featured = data?.events ?? [];

  return (
    <div>
      {/* Hero */}
      <section className="relative overflow-hidden">
        <div className="pointer-events-none absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-yellow-950/30 via-background to-background" />
          <div className="absolute -top-24 left-1/2 size-[40rem] -translate-x-1/2 rounded-full bg-yellow-500/10 blur-3xl" />
          <div className="absolute -bottom-24 right-1/4 size-[30rem] -translate-x-1/2 rounded-full bg-pink-500/5 blur-3xl" />
        </div>

        <PageContainer className="flex min-h-[88vh] flex-col items-center justify-center py-24 text-center">
          <motion.span
            custom={0}
            variants={fadeUp}
            initial="hidden"
            animate="show"
            className="inline-flex items-center gap-2 rounded-full border bg-card/70 px-3 py-1 text-xs font-medium text-muted-foreground backdrop-blur"
          >
            <Sparkles className="size-3.5 text-primary" />
            Delightful events, end to end
          </motion.span>

          <motion.h1
            custom={1}
            variants={fadeUp}
            initial="hidden"
            animate="show"
            className="mt-6 max-w-4xl text-balance text-5xl font-bold leading-[1.05] tracking-tight sm:text-6xl md:text-7xl"
          >
            Delightful events
            <br />
            <span className="text-gradient">
              start here.
            </span>
          </motion.h1>

          <motion.p
            custom={2}
            variants={fadeUp}
            initial="hidden"
            animate="show"
            className="mt-6 max-w-xl text-pretty text-lg text-muted-foreground"
          >
            Set up an event page, invite friends, and sell tickets. Host a
            memorable event today — or discover one near you.
          </motion.p>

          <motion.div
            custom={3}
            variants={fadeUp}
            initial="hidden"
            animate="show"
            className="mt-8 flex flex-col items-center gap-3 sm:flex-row"
          >
            <Button asChild size="lg" className="rounded-full px-6 text-base">
              <Link to="/create">
                Create your first event
                <ArrowRight data-icon="inline-end" />
              </Link>
            </Button>
            <Button asChild size="lg" variant="outline" className="rounded-full px-6 text-base">
              <Link to="/discover">Discover events</Link>
            </Button>
          </motion.div>
        </PageContainer>
      </section>

      {/* Featured upcoming events */}
      <section className="py-16 sm:py-20">
        <PageContainer>
          <div className="flex items-end justify-between gap-4">
            <div>
              <h2 className="text-2xl font-bold tracking-tight sm:text-3xl">Upcoming events</h2>
              <p className="mt-1 text-muted-foreground">Fresh events looking for attendees.</p>
            </div>
            <Button asChild variant="ghost" className="hidden sm:inline-flex">
              <Link to="/discover">
                Browse all
                <ArrowRight data-icon="inline-end" />
              </Link>
            </Button>
          </div>

          {isLoading ? (
            <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
              {Array.from({ length: 8 }).map((_, i) => (
                <EventCardSkeleton key={i} />
              ))}
            </div>
          ) : isError ? (
            <div className="mt-8 flex flex-col items-center gap-3 rounded-2xl border border-dashed p-10 text-center">
              <p className="text-muted-foreground">Couldn't load upcoming events.</p>
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                Try again
              </Button>
            </div>
          ) : featured.length > 0 ? (
            <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
              {featured.map((event, i) => (
                <EventCard key={event.id} event={event} index={i} />
              ))}
            </div>
          ) : (
            <p className="mt-8 rounded-2xl border border-dashed p-10 text-center text-muted-foreground">
              No upcoming events yet — be the first to host one.
            </p>
          )}
        </PageContainer>
      </section>

      {/* Browse by category */}
      <section className="bg-muted/40 py-16 sm:py-20">
        <PageContainer>
          <h2 className="text-2xl font-bold tracking-tight sm:text-3xl">Browse by category</h2>
          <p className="mt-1 text-muted-foreground">Find something you love.</p>
          <div className="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
            {CATEGORIES.map((c) => (
              <Link
                key={c}
                to={`/discover?category=${c}`}
                className="group flex items-center justify-between rounded-2xl border bg-card p-5 transition-all hover:-translate-y-0.5 hover:shadow-md"
              >
                <div>
                  <p className="font-semibold">{CATEGORY_LABELS[c]}</p>
                  <p className="text-sm text-muted-foreground">Explore</p>
                </div>
                <ArrowRight className="size-5 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-primary" />
              </Link>
            ))}
          </div>
        </PageContainer>
      </section>

      {/* How it works */}
      <section className="py-16 sm:py-20">
        <PageContainer>
          <h2 className="text-center text-2xl font-bold tracking-tight sm:text-3xl">
            Everything in one place
          </h2>
          <div className="mt-10 grid gap-6 md:grid-cols-3">
            <Step
              icon={<Sparkles className="size-6" />}
              title="Discover"
              body="Browse curated events by category, date, or format. Virtual or in person."
            />
            <Step
              icon={<Ticket className="size-6" />}
              title="Register"
              body="Grab a ticket in seconds. Get a personal access token and a shareable ticket page."
            />
            <Step
              icon={<CalendarCheck className="size-6" />}
              title="Host"
              body="Create a page, manage approvals, and check attendees in at the door."
            />
          </div>
        </PageContainer>
      </section>

      {/* CTA band */}
      <section className="px-4 pb-24 sm:px-6">
        <PageContainer>
          <div className="relative overflow-hidden rounded-3xl bg-foreground px-8 py-16 text-center text-background sm:py-20">
            <div className="pointer-events-none absolute -right-16 -top-16 size-64 rounded-full bg-primary/30 blur-3xl" />
            <h2 className="relative text-3xl font-bold tracking-tight sm:text-4xl">
              Ready to host?
            </h2>
            <p className="relative mx-auto mt-3 max-w-md text-background/80">
              Set up your event page for free and open registrations in minutes.
            </p>
            <Button asChild size="lg" className="relative mt-8 rounded-full px-6 text-base">
              <Link to="/create">
                <MapPinned className="size-4" data-icon="inline-start" />
                Create an event
              </Link>
            </Button>
          </div>
        </PageContainer>
      </section>
    </div>
  );
}

function Step({ icon, title, body }: { icon: React.ReactNode; title: string; body: string }) {
  return (
    <div className="rounded-2xl border bg-card p-6">
      <div className="grid size-11 place-items-center rounded-xl bg-primary/10 text-primary">
        {icon}
      </div>
      <h3 className="mt-4 text-lg font-semibold">{title}</h3>
      <p className="mt-1.5 text-sm text-muted-foreground">{body}</p>
    </div>
  );
}
