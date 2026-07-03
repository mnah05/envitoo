import { Link, useParams } from "react-router-dom";
import { useCallback, useEffect, useState } from "react";
import { ArrowLeft, CalendarDays, Check, Clock, Globe, MapPin, Share2, Users } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { PageContainer } from "@/components/domain/Layout";
import { EventCover } from "@/components/domain/EventCard";
import { CategoryPill } from "@/components/domain/CategoryPill";
import { EventStatusBadge } from "@/components/domain/StatusBadge";
import { RegisterForm } from "@/components/domain/RegisterForm";
import { VolunteerApplyForm } from "@/components/domain/VolunteerApplyForm";
import { EventMap } from "@/components/domain/EventMap";
import { useEventBySlug } from "@/hooks/use-events";
import { useVolunteerSettings } from "@/hooks/use-volunteers";
import { formatEventDate, formatFullDate, formatPrice } from "@/lib/format";

export function EventDetail() {
  const { slug = "" } = useParams();
  const { data: event, isLoading, error } = useEventBySlug(slug);
  const { data: volunteerSettings } = useVolunteerSettings(event?.id ?? "", !!event);

  useEffect(() => {
    document.title = event ? `${event.title} — EnviToo` : "Event — EnviToo";
  }, [event]);

  const eventUrl = event ? `${window.location.origin}/e/${event.slug}` : "";
  const [copied, setCopied] = useState(false);

  const share = useCallback(async () => {
    if (!eventUrl) return;
    try {
      if (typeof navigator.share === "function") {
        await navigator.share({ title: event?.title ?? "EnviToo event", url: eventUrl });
        return;
      }
      await navigator.clipboard.writeText(eventUrl);
      toast.success("Link copied!");
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      if (typeof navigator.share !== "function") toast.error("Could not copy link");
    }
  }, [eventUrl, event?.title]);

  if (isLoading) {
    return <EventDetailSkeleton />;
  }

  if (error || !event) {
    return (
      <div className="pt-16">
        <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
          <h1 className="text-2xl font-bold">Event not found</h1>
          <p className="text-muted-foreground">
            This event may have been cancelled or doesn't exist.
          </p>
          <Button asChild variant="outline">
            <Link to="/discover">Discover events</Link>
          </Button>
        </PageContainer>
      </div>
    );
  }

  const cancelled = event.status === "cancelled";

  return (
    <div className="pt-16">
      {/* Cover */}
      <div className="relative h-[28vh] min-h-[14rem] w-full overflow-hidden sm:h-[40vh] sm:min-h-[20rem]">
        <EventCover event={event} rounded="rounded-none" className="h-full" />
        <div className="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent" />
        <PageContainer className="absolute inset-x-0 bottom-0">
          <div className="flex flex-wrap items-center gap-2 pb-6">
            {event.category && <CategoryPill category={event.category} />}
            <EventStatusBadge status={event.status} />
            {event.visibility === "unlisted" && (
              <span className="inline-flex items-center gap-1.5 rounded-full border border-dashed bg-card/80 px-2.5 py-0.5 text-xs font-medium text-muted-foreground backdrop-blur">
                Unlisted — not on Discover
              </span>
            )}
            {event.is_virtual && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-white/90 px-2.5 py-0.5 text-xs font-medium text-foreground backdrop-blur">
                <Globe className="size-3.5" /> Virtual
              </span>
            )}
          </div>
        </PageContainer>
      </div>

      <PageContainer className="py-10">
        <div className="mb-6 flex items-center justify-between gap-2">
          <Button asChild variant="ghost" size="sm" className="-ml-2">
            <Link to="/discover">
              <ArrowLeft data-icon="inline-start" />
              Back to discover
            </Link>
          </Button>
          <Button variant="outline" size="sm" onClick={share} className="gap-1.5">
            {copied ? (
              <>
                <Check className="size-4" /> Copied!
              </>
            ) : (
              <>
                <Share2 className="size-4" /> Share
              </>
            )}
          </Button>
        </div>

        <div className="flex flex-col gap-10 lg:grid lg:grid-cols-[1fr_22rem]">
          {/* Main */}
          <div className="flex flex-col gap-6">
            <div>
              <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">{event.title}</h1>
              <p className="mt-3 text-muted-foreground">{formatFullDate(event.starts_at)}</p>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <InfoRow icon={<CalendarDays className="size-5" />} label="When">
                {formatEventDate(event.starts_at, event.ends_at)}
              </InfoRow>
              <InfoRow
                icon={event.is_virtual ? <Globe className="size-5" /> : <MapPin className="size-5" />}
                label="Where"
              >
                {event.is_virtual ? (
                  <a
                    href={event.meeting_url ?? "#"}
                    target="_blank"
                    rel="noreferrer"
                    className="text-primary hover:underline"
                  >
                    {event.meeting_url ? "Join online" : "Virtual event"}
                  </a>
                ) : event.venue_name || event.location_address ? (
                  <span>
                    {event.venue_name && (
                      <span className="block font-medium">{event.venue_name}</span>
                    )}
                    {event.location_address && (
                      <span className="text-muted-foreground">{event.location_address}</span>
                    )}
                  </span>
                ) : (
                  "Location to be announced"
                )}
              </InfoRow>
              <InfoRow icon={<Users className="size-5" />} label="Capacity">
                {event.capacity} seats
                {!cancelled && (
                  <span className="block text-sm font-normal text-muted-foreground">
                    {Math.max(0, event.capacity - event.registration_count)} seats left
                  </span>
                )}
              </InfoRow>
              <InfoRow icon={<Clock className="size-5" />} label="Price">
                {formatPrice(event.price_in_paise, event.currency)}
              </InfoRow>
            </div>
          </div>

          {/* Register + volunteer card — above description on mobile, right column on desktop */}
          <aside className="flex flex-col gap-6 lg:sticky lg:top-24 lg:row-span-2 lg:self-start">
            <div className="rounded-2xl border bg-card p-6 shadow-sm">
              {cancelled ? (
                <div className="rounded-xl bg-muted p-6 text-center">
                  <p className="font-semibold">This event has been cancelled.</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Registrations are closed.
                  </p>
                </div>
              ) : (
                <>
                  <div className="mb-4 flex items-baseline justify-between">
                    <span className="text-2xl font-bold">
                      {formatPrice(event.price_in_paise, event.currency)}
                    </span>
                    {event.requires_approval && (
                      <span className="rounded-full bg-amber-50 px-2.5 py-0.5 text-xs font-medium text-amber-700 ring-1 ring-inset ring-amber-200">
                        Approval required
                      </span>
                    )}
                  </div>
                  <RegisterForm event={event} />
                </>
              )}
            </div>

            {volunteerSettings?.enabled && !cancelled && (
              <VolunteerApplyForm event={event} settings={volunteerSettings} />
            )}
          </aside>

          {/* Map — physical events with coordinates */}
          {!event.is_virtual && event.lat != null && event.lng != null && (
            <div className="lg:col-start-1 lg:row-start-3">
              <h2 className="mb-3 text-lg font-semibold">Where to find it</h2>
              <EventMap lat={event.lat} lng={event.lng} label={event.venue_name ?? event.title} />
            </div>
          )}

          {/* Description — below register card on mobile, below info on desktop */}
          {event.description && (
            <div className="flex flex-col gap-3 lg:col-start-1 lg:row-start-2">
              <h2 className="text-lg font-semibold">About this event</h2>
              <Separator />
              <p className="whitespace-pre-line text-pretty leading-relaxed text-muted-foreground">
                {event.description}
              </p>
            </div>
          )}
        </div>
      </PageContainer>
    </div>
  );
}

function InfoRow({
  icon,
  label,
  children,
}: {
  icon: React.ReactNode;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-start gap-3 rounded-xl border bg-card p-4">
      <span className="mt-0.5 text-muted-foreground">{icon}</span>
      <div className="min-w-0">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </p>
        <p className="mt-0.5 font-medium">{children}</p>
      </div>
    </div>
  );
}

function EventDetailSkeleton() {
  return (
    <div className="pt-16">
      <Skeleton className="h-[40vh] min-h-[20rem] w-full rounded-none" />
      <PageContainer className="py-10">
        <Skeleton className="mb-6 h-8 w-24" />
        <div className="grid gap-10 lg:grid-cols-[1fr_22rem]">
          <div className="flex flex-col gap-6">
            <div className="flex flex-col gap-3">
              <Skeleton className="h-9 w-2/3" />
              <Skeleton className="h-4 w-1/3" />
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="flex items-start gap-3 rounded-xl border bg-card p-4">
                  <Skeleton className="size-5 rounded" />
                  <div className="flex flex-col gap-2">
                    <Skeleton className="h-3 w-12" />
                    <Skeleton className="h-4 w-32" />
                  </div>
                </div>
              ))}
            </div>
            <div className="flex flex-col gap-3">
              <Skeleton className="h-6 w-40" />
              <Skeleton className="h-px w-full" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-5/6" />
              <Skeleton className="h-4 w-2/3" />
            </div>
          </div>
          <aside className="flex flex-col gap-6 lg:sticky lg:top-24 lg:self-start">
            <div className="rounded-2xl border bg-card p-6 shadow-sm">
              <div className="mb-4 flex items-baseline justify-between">
                <Skeleton className="h-8 w-24" />
                <Skeleton className="h-5 w-24 rounded-full" />
              </div>
              <div className="flex flex-col gap-4">
                <Skeleton className="h-9 w-full" />
                <Skeleton className="h-9 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            </div>
          </aside>
        </div>
      </PageContainer>
    </div>
  );
}
