import { Link, useParams } from "react-router-dom";
import { useCallback, useState } from "react";
import { ArrowLeft, CalendarDays, Check, Clock, Globe, MapPin, Share2, Users } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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
import { usePageMeta } from "@/hooks/use-page-meta";
import { formatEventDate, formatFullDate, formatPrice } from "@/lib/format";

export function EventDetail() {
  const { slug = "" } = useParams();
  const { data: event, isLoading, error } = useEventBySlug(slug);
  const { data: volunteerSettings } = useVolunteerSettings(event?.id ?? "", !!event);

  const description = event?.description
    ? event.description.replace(/\s+/g, " ").trim().slice(0, 200)
    : undefined;

  usePageMeta({
    title: event ? `${event.title} — EnviToo` : "Event — EnviToo",
    description,
    image: event?.cover_image_url ?? undefined,
    url: event ? `/e/${event.slug}` : undefined,
    type: "event",
  });

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
      <div className="pt-14">
        <PageContainer className="flex min-h-[60vh] flex-col items-center justify-center gap-4 text-center">
          <h1 className="display-title text-2xl">Event not found</h1>
          <p className="max-w-sm text-base leading-relaxed text-muted-foreground">
            This event may have been cancelled or doesn&apos;t exist.
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
    <div className="pt-14">
      <div className="relative h-[28vh] min-h-[14rem] w-full overflow-hidden sm:h-[40vh] sm:min-h-[20rem]">
        <EventCover event={event} rounded="rounded-none" className="h-full" />
        <div className="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent" />
        <PageContainer className="absolute inset-x-0 bottom-0">
          <div className="flex flex-wrap items-center gap-2 pb-6">
            {event.category && <CategoryPill category={event.category} />}
            <EventStatusBadge status={event.status} />
            {event.visibility === "unlisted" && (
              <span className="inline-flex items-center gap-1.5 rounded-md border border-dashed border-border bg-card/80 px-2 py-0.5 text-xs font-medium text-muted-foreground backdrop-blur">
                Unlisted — not on Discover
              </span>
            )}
            {event.is_virtual && (
              <span className="inline-flex items-center gap-1.5 rounded-md bg-foreground/10 px-2 py-0.5 text-xs font-medium text-foreground/80 backdrop-blur">
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
          <div className="flex flex-col gap-8">
            <div>
              <h1 className="display-title text-3xl sm:text-4xl">{event.title}</h1>
              <p className="mt-3 text-base leading-relaxed text-muted-foreground">
                {formatFullDate(event.starts_at)}
              </p>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
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

            {event.description && (
              <div className="flex flex-col gap-3">
                <h2 className="display-headline-sans text-base font-semibold">About this event</h2>
                <p className="whitespace-pre-line text-pretty text-sm leading-relaxed text-muted-foreground">
                  {event.description}
                </p>
              </div>
            )}

            {!event.is_virtual && event.lat != null && event.lng != null && (
              <div>
                <h2 className="display-headline-sans mb-3 text-base font-semibold">Where to find it</h2>
                <EventMap lat={event.lat} lng={event.lng} label={event.venue_name ?? event.title} />
              </div>
            )}
          </div>

          <aside className="flex flex-col gap-6 lg:sticky lg:top-24 lg:row-span-2 lg:self-start">
            <div className="rounded-xl border border-self-strong bg-card p-6">
              {cancelled ? (
                <div className="rounded-lg bg-muted p-6 text-center">
                  <p className="display-headline text-base">This event has been cancelled.</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Registrations are closed.
                  </p>
                </div>
              ) : (
                <>
                  <div className="mb-4 flex items-baseline justify-between">
                    <span className="text-stat text-2xl">
                      {formatPrice(event.price_in_paise, event.currency)}
                    </span>
                    {event.requires_approval && (
                      <span className="rounded-md bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-400">
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
    <div className="flex items-start gap-3">
      <span className="mt-0.5 text-muted-foreground">{icon}</span>
      <div className="min-w-0">
        <p className="label-muted text-[12px]">{label}</p>
        <p className="mt-0.5 text-sm font-medium">{children}</p>
      </div>
    </div>
  );
}

function EventDetailSkeleton() {
  return (
    <div className="pt-14">
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
                <div key={i} className="flex items-start gap-3">
                  <Skeleton className="mt-0.5 size-5 rounded" />
                  <div className="flex flex-col gap-2">
                    <Skeleton className="h-3 w-12" />
                    <Skeleton className="h-4 w-32" />
                  </div>
                </div>
              ))}
            </div>
          </div>
          <aside className="flex flex-col gap-6 lg:sticky lg:top-24 lg:self-start">
            <div className="rounded-xl border bg-card p-6">
              <Skeleton className="mb-4 h-8 w-24" />
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
