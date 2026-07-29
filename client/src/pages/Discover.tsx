import { Button } from "@/components/ui/button";
import { useEffect, useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { useReducedMotion } from "framer-motion";
import { CompassIcon, SparklesIcon } from "lucide-animated";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { CategoryIcon } from "@/components/domain/CategoryIcon";
import { Empty, EmptyDescription, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS, type Event } from "@/lib/api";

const PAGE_SIZE = 100;

export function Discover() {
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedCategory = searchParams.get("category");
  const reduceMotion = useReducedMotion();

  const nowIso = useMemo(() => new Date().toISOString(), []);

  const { data, isLoading, isError, refetch } = usePublicEvents({
    startsAfter: nowIso,
    limit: PAGE_SIZE,
  });

  const allEvents = data?.events ?? [];

  const published = useMemo(
    () =>
      allEvents.filter(
        (e) =>
          e.status === "published" &&
          e.registration_count < e.capacity &&
          (!selectedCategory || (e.category ?? "other") === selectedCategory),
      ),
    [allEvents, selectedCategory],
  );

  const grouped = useMemo(() => {
    const map = new Map<string, typeof allEvents>();
    for (const e of allEvents) {
      if (e.status !== "published") continue;
      if (e.registration_count >= e.capacity) continue;
      const cat = e.category ?? "other";
      if (!map.has(cat)) map.set(cat, []);
      map.get(cat)!.push(e);
    }
    return map;
  }, [allEvents]);

  const activeCategories = CATEGORIES.filter((c) => (grouped.get(c)?.length ?? 0) > 0);

  useEffect(() => {
    document.title = "Discover — EnviToo";
  }, []);

  function clearCategory() {
    setSearchParams((prev) => {
      prev.delete("category");
      return prev;
    });
  }

  return (
    <div className="pt-14">
      <PageContainer className="py-12 sm:py-16">
        <h1 className="display-title text-4xl sm:text-5xl">Discover events</h1>
        <p className="mt-4 max-w-lg text-base leading-relaxed text-muted-foreground">
          What&apos;s happening next — find your next experience.
        </p>
      </PageContainer>

      <PageContainer className="pb-6">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          {CATEGORIES.map((c) => {
            const isActive = selectedCategory === c;
            return (
              <Link
                key={c}
                to={`/discover?category=${c}`}
                className={
                  "group flex items-center gap-3 rounded-lg border p-4 text-sm font-medium transition-colors" +
                  (isActive
                    ? " border-primary/40 bg-primary/10 text-foreground"
                    : " border-border bg-card text-muted-foreground hover:border-primary/30 hover:text-foreground")
                }
              >
                <span className="flex size-8 items-center justify-center rounded-md bg-primary/10 text-primary" aria-hidden>
                  <CategoryIcon category={c} animateOnHover={!reduceMotion} />
                </span>
                <span>{CATEGORY_LABELS[c]}</span>
              </Link>
            );
          })}
        </div>
      </PageContainer>

      <PageContainer className="pb-28 sm:pb-20">
        {selectedCategory && (
          <div className="mb-8 flex items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              Showing{" "}
              <span className="font-medium text-foreground">
                {CATEGORY_LABELS[selectedCategory as keyof typeof CATEGORY_LABELS] ?? selectedCategory}
              </span>{" "}
              events
            </p>
            <button
              type="button"
              onClick={clearCategory}
              className="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              All categories &rarr;
            </button>
          </div>
        )}

        {isError ? (
          <Empty className="min-h-[40vh]">
            <EmptyMedia variant="icon">
              <CompassIcon size={24} animateOnHover={!reduceMotion} />
            </EmptyMedia>
            <EmptyTitle>Couldn&apos;t load events</EmptyTitle>
            <EmptyDescription>
              Check your connection and try again.
            </EmptyDescription>
            <Button variant="outline" onClick={() => refetch()}>
              Retry
            </Button>
          </Empty>
        ) : isLoading ? (
          <div className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <EventCardSkeleton key={i} />
            ))}
          </div>
        ) : selectedCategory ? (
          published.length > 0 ? (
            <div className="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {published.map((event, i) => (
                <EventCard key={event.id} event={event} index={i} />
              ))}
            </div>
          ) : (
            <Empty className="min-h-[40vh]">
              <EmptyMedia variant="icon">
                <SparklesIcon size={24} animateOnHover={!reduceMotion} />
              </EmptyMedia>
              <EmptyTitle>No events in this category</EmptyTitle>
              <EmptyDescription>
                Check back soon — new events are being added every day.
              </EmptyDescription>
              <Button variant="outline" onClick={clearCategory}>
                View all categories
              </Button>
            </Empty>
          )
        ) : activeCategories.length === 0 ? (
          <Empty className="min-h-[40vh]">
            <EmptyMedia variant="icon">
              <SparklesIcon size={24} animateOnHover={!reduceMotion} />
            </EmptyMedia>
            <EmptyTitle>No events yet</EmptyTitle>
            <EmptyDescription>
              Check back soon — new events are being added every day.
            </EmptyDescription>
          </Empty>
        ) : (
          <div className="mt-10 space-y-14">
            {activeCategories.map((cat) => (
              <CategorySection
                key={cat}
                label={CATEGORY_LABELS[cat]}
                events={grouped.get(cat)!}
              />
            ))}
          </div>
        )}
      </PageContainer>
    </div>
  );
}

function CategorySection({
  label,
  events,
}: {
  label: string;
  events: Event[];
}) {
  return (
    <section>
      <h2 className="display-headline-sans mb-4 text-lg font-semibold">{label}</h2>
      <div className="relative">
        <div className="flex gap-4 overflow-x-auto pb-2 scrollbar-none">
          {events.map((event, i) => (
            <div key={event.id} className="min-w-[280px] max-w-[320px] flex-shrink-0">
              <EventCard event={event} index={i} />
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
