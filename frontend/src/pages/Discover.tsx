import { Button } from "@/components/ui/button";
import { useEffect, useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS, type Event } from "@/lib/api";

const PAGE_SIZE = 100;

export function Discover() {
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedCategory = searchParams.get("category");

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

  // Group non-full published events by category
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

  // Ordered categories that have events
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
    <div className="pt-16">
      <PageContainer className="py-14 sm:py-20">
        <p className="text-eyebrow text-primary/90">Browse</p>
        <h1 className="display-hero mt-3 text-4xl sm:text-5xl">
          Discover events
        </h1>
        <p className="text-lede mt-4 max-w-lg text-[1.0625rem]">
          What's happening next — find your next experience.
        </p>
      </PageContainer>

      <PageContainer className="pb-6">
        <h2 className="label-muted mb-4">Browse by category</h2>
        <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-none sm:grid sm:grid-cols-3 sm:overflow-visible lg:grid-cols-4">
          {CATEGORIES.map((c) => {
            const isActive = selectedCategory === c;
            return (
              <Link
                key={c}
                to={`/discover?category=${c}`}
                className={
                  "group flex min-w-[14rem] flex-shrink-0 items-center justify-between gap-2 rounded-xl border border-border/80 bg-card p-4 transition-colors hover:border-primary/35 hover:bg-primary/5 sm:min-w-0" +
                  (isActive ? " border-primary/40 bg-primary/10" : "")
                }
              >
                <div>
                  <p className="text-sm font-medium">{CATEGORY_LABELS[c]}</p>
                  <p className="text-xs text-muted-foreground">Explore</p>
                </div>
                <ArrowRight
                  className={
                    "size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-foreground" +
                    (isActive ? " text-foreground" : "")
                  }
                />
              </Link>
            );
          })}
        </div>
      </PageContainer>

      {/* Content */}
      <PageContainer className="pb-28 sm:pb-20">
        {selectedCategory && (
          <div className="mb-8 flex items-center justify-between gap-3">
            <p className="text-muted-foreground">
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
              All categories →
            </button>
          </div>
        )}

        {isError ? (
          <Empty className="min-h-[40vh] border">
            <EmptyTitle>Couldn't load events</EmptyTitle>
            <EmptyDescription>
              Check your connection and try again.
            </EmptyDescription>
            <Button variant="outline" onClick={() => refetch()}>
              Retry
            </Button>
          </Empty>
        ) : isLoading ? (
          <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <EventCardSkeleton key={i} />
            ))}
          </div>
        ) : selectedCategory ? (
          published.length > 0 ? (
            <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {published.map((event, i) => (
                <EventCard key={event.id} event={event} index={i} />
              ))}
            </div>
          ) : (
            <Empty className="min-h-[40vh] border">
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
          <Empty className="min-h-[40vh] border">
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
      <h2 className="display-headline mb-4 text-xl">{label}</h2>
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