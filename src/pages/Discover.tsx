import { useEffect, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PageContainer } from "@/components/domain/Layout";
import { EventCard } from "@/components/domain/EventCard";
import { EventCardSkeleton } from "@/components/domain/EventCardSkeleton";
import { Empty, EmptyDescription, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { usePublicEvents } from "@/hooks/use-events";
import { CATEGORIES, CATEGORY_LABELS } from "@/lib/api";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 12;

export function Discover() {
  const [params, setParams] = useSearchParams();
  const category = params.get("category") ?? "";
  const q = params.get("q") ?? "";
  const format = params.get("format") ?? "all"; // all | virtual | physical
  const offset = Number(params.get("offset") ?? "0") || 0;

  const isVirtual = format === "virtual" ? true : format === "physical" ? false : undefined;
  const nowIso = useMemo(() => new Date().toISOString(), []);

  const { data, isLoading, isFetching, isError, refetch } = usePublicEvents({
    category: category || undefined,
    q: q || undefined,
    isVirtual,
    startsAfter: nowIso,
    limit: PAGE_SIZE,
    offset,
  });

  const events = data?.events ?? [];
  const total = data?.total ?? 0;
  const hasMore = offset + events.length < total;

  function update(next: Record<string, string | null>) {
    const merged = new URLSearchParams(params);
    for (const [k, v] of Object.entries(next)) {
      if (v === null || v === "") merged.delete(k);
      else merged.set(k, v);
    }
    merged.delete("offset"); // reset paging on filter change
    setParams(merged, { replace: true });
  }

  // Keep debounced search input in sync with a local state for snappy typing.
  // For simplicity we update the URL on change; TanStack keepPreviousData
  // smooths the transitions.
  useEffect(() => {
    document.title = "Discover events — Pulse";
  }, []);

  return (
    <div className="pt-16">
      {/* Header */}
      <div className="border-b bg-background">
        <PageContainer className="py-10">
          <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">Discover events</h1>
          <p className="mt-2 text-muted-foreground">
            Explore popular events, browse by category, or search for something specific.
          </p>

          <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={q}
                onChange={(e) => update({ q: e.target.value })}
                placeholder="Search events"
                className="rounded-full pl-9"
              />
            </div>
            <div className="flex gap-1 rounded-full bg-muted p-1">
              {(["all", "virtual", "physical"] as const).map((f) => (
                <button
                  key={f}
                  onClick={() => update({ format: f === "all" ? null : f })}
                  className={cn(
                    "rounded-full px-3 py-1.5 text-sm font-medium capitalize transition-colors",
                    format === f ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  {f}
                </button>
              ))}
            </div>
          </div>
        </PageContainer>
      </div>

      {/* Category pills */}
      <PageContainer className="py-6">
        <div className="flex flex-wrap gap-2">
          <CategoryButton active={category === ""} onClick={() => update({ category: null })}>
            All
          </CategoryButton>
          {CATEGORIES.map((c) => (
            <CategoryButton key={c} active={category === c} onClick={() => update({ category: c })}>
              {CATEGORY_LABELS[c]}
            </CategoryButton>
          ))}
        </div>
      </PageContainer>

      {/* Grid */}
      <PageContainer className="pb-20">
        {isError ? (
          <Empty className="min-h-[40vh] border">
            <EmptyMedia variant="icon">
              <Search className="size-6" />
            </EmptyMedia>
            <EmptyTitle>Couldn't load events</EmptyTitle>
            <EmptyDescription>
              Check your connection and try again.
            </EmptyDescription>
            <Button variant="outline" onClick={() => refetch()}>
              Retry
            </Button>
          </Empty>
        ) : isLoading && events.length === 0 ? (
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <EventCardSkeleton key={i} />
            ))}
          </div>
        ) : events.length === 0 ? (
          <Empty className="min-h-[40vh] border">
            <EmptyMedia variant="icon">
              <Search className="size-6" />
            </EmptyMedia>
            <EmptyTitle>No events found</EmptyTitle>
            <EmptyDescription>
              Try a different category, format, or search term.
            </EmptyDescription>
          </Empty>
        ) : (
          <>
            <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {events.map((event, i) => (
                <EventCard key={event.id} event={event} index={i} />
              ))}
            </div>
            {hasMore && (
              <div className="mt-10 flex justify-center">
                <Button
                  variant="outline"
                  size="lg"
                  className="rounded-full"
                  disabled={isFetching}
                  onClick={() =>
                    setParams(
                      (prev) => {
                        const next = new URLSearchParams(prev);
                        next.set("offset", String(offset + PAGE_SIZE));
                        return next;
                      },
                      { replace: true },
                    )
                  }
                >
                  {isFetching ? "Loading…" : "Load more"}
                </Button>
              </div>
            )}
          </>
        )}
      </PageContainer>
    </div>
  );
}

function CategoryButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "rounded-full border px-3.5 py-1.5 text-sm font-medium transition-colors",
        active
          ? "border-primary bg-primary text-primary-foreground"
          : "border-border bg-card text-muted-foreground hover:border-foreground/20 hover:text-foreground",
      )}
    >
      {children}
    </button>
  );
}
