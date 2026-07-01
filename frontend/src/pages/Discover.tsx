import { Button } from "@/components/ui/button";
import { useEffect, useMemo } from "react";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS, type Event } from "@/lib/api";

const PAGE_SIZE = 100;

export function Discover() {
  const nowIso = useMemo(() => new Date().toISOString(), []);

  const { data, isLoading, isError, refetch } = usePublicEvents({
    startsAfter: nowIso,
    limit: PAGE_SIZE,
  });

  const allEvents = data?.events ?? [];

  // Group non-full events by category
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

  return (
    <div className="pt-16">
      {/* Hero */}
      <div className="border-b bg-background">
        <PageContainer className="py-10">
          <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Discover events
          </h1>
          <p className="mt-2 text-muted-foreground">
            Find your next experience — from tech talks to fitness meetups.
          </p>
        </PageContainer>
      </div>

      {/* Content */}
      <PageContainer className="pb-20">
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
          <div className="mt-10 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <EventCardSkeleton key={i} />
            ))}
          </div>
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
      <h2 className="mb-4 text-xl font-semibold">{label}</h2>
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
