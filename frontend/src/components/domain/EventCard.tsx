import { Link } from "react-router-dom";
import { Calendar, MapPin, Monitor } from "lucide-react";
import { cn } from "@/lib/utils";
import { formatCardDate, formatCardTime, formatPrice } from "@/lib/format";
import { type Event } from "@/lib/api";

export function EventCard({ event, index = 0 }: { event: Event; index?: number }) {
  const href = `/e/${event.slug}`;
  return (
    <Link
      to={href}
      className="group flex flex-col overflow-hidden rounded-xl border border-border/80 bg-card transition-colors hover:border-primary/35 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      style={{ animationDelay: `${index * 40}ms` }}
    >
      <EventCover event={event} rounded="rounded-none" />
      <div className="flex flex-1 flex-col gap-2 p-4">
        <span className="text-eyebrow text-[12px]">
          {formatCardDate(event.starts_at)}
        </span>
        <h3 className="display-headline line-clamp-2 text-[1.0625rem] leading-snug text-foreground transition-colors group-hover:text-foreground/90">
          {event.title}
        </h3>
        <div className="mt-auto flex items-center gap-1.5 pt-2 text-[13px] tracking-[-0.01em] text-muted-foreground">
          {event.is_virtual ? (
            <Monitor className="size-3.5 shrink-0" />
          ) : (
            <MapPin className="size-3.5 shrink-0" />
          )}
          <span className="truncate">
            {event.is_virtual ? "Virtual event" : event.venue_name || "Location"}
          </span>
        </div>
        <div className="flex items-center justify-between pt-1">
          <span className="flex items-center gap-1.5 text-[13px] tracking-[-0.01em] text-muted-foreground">
            <Calendar className="size-3.5 shrink-0" />
            {formatCardTime(event.starts_at)}
          </span>
          <span className="text-ui text-[13px] font-semibold tabular-nums text-foreground">
            {formatPrice(event.price_in_paise, event.currency)}
          </span>
        </div>
      </div>
    </Link>
  );
}

/** Cover image with a deterministic gradient fallback when none is set. */
export function EventCover({
  event,
  className,
  rounded = "rounded-2xl",
}: {
  event: Event;
  className?: string;
  rounded?: string;
}) {
  return (
    <div
      className={cn(
        "relative aspect-[16/9] w-full overflow-hidden bg-muted",
        rounded,
        className,
      )}
    >
      {event.cover_image_url ? (
        <img
          src={event.cover_image_url}
          alt=""
          loading="lazy"
          className="size-full object-cover"
        />
      ) : (
        <div className={cn("size-full", gradientFor(event.id))} />
      )}
    </div>
  );
}

const GRADIENTS = [
  "bg-gradient-to-br from-teal-500 via-teal-800 to-zinc-950",
  "bg-gradient-to-br from-cyan-500 via-sky-900 to-zinc-950",
  "bg-gradient-to-br from-rose-500 via-rose-900 to-zinc-950",
  "bg-gradient-to-br from-emerald-500 via-emerald-900 to-zinc-950",
  "bg-gradient-to-br from-amber-500 via-orange-800 to-zinc-950",
  "bg-gradient-to-br from-sky-500 via-blue-900 to-zinc-950",
];

function gradientFor(id: string): string {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0;
  return GRADIENTS[h % GRADIENTS.length];
}
