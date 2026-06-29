import { Link } from "react-router-dom";
import { Calendar, MapPin, Monitor } from "lucide-react";
import { cn } from "@/lib/utils";
import { formatCardDate, formatCardTime, formatPrice } from "@/lib/format";
import { type Event } from "@/lib/api";
import { CategoryPill } from "./CategoryPill";

export function EventCard({ event, index = 0 }: { event: Event; index?: number }) {
  const href = `/e/${event.slug}`;
  return (
    <Link
      to={href}
      className="group flex flex-col overflow-hidden rounded-2xl border bg-card transition-all hover:-translate-y-0.5 hover:shadow-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      style={{ animationDelay: `${index * 40}ms` }}
    >
      <EventCover event={event} />
      <div className="flex flex-1 flex-col gap-2 p-4">
        <div className="flex items-center gap-2">
          {event.category && <CategoryPill category={event.category} />}
          <span className="text-xs font-medium text-muted-foreground">
            {formatCardDate(event.starts_at)}
          </span>
        </div>
        <h3 className="line-clamp-2 text-base font-semibold leading-snug text-foreground group-hover:text-primary">
          {event.title}
        </h3>
        <div className="mt-auto flex items-center gap-1.5 pt-2 text-sm text-muted-foreground">
          {event.is_virtual ? (
            <Monitor className="size-4 shrink-0" />
          ) : (
            <MapPin className="size-4 shrink-0" />
          )}
          <span className="truncate">
            {event.is_virtual ? "Virtual event" : event.venue_name || "Location"}
          </span>
        </div>
        <div className="flex items-center justify-between pt-1">
          <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <Calendar className="size-4 shrink-0" />
            {formatCardTime(event.starts_at)}
          </span>
          <span className="text-sm font-semibold text-foreground">
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
          className="size-full object-cover transition-transform duration-500 group-hover:scale-105"
        />
      ) : (
        <div className={cn("size-full", gradientFor(event.id))} />
      )}
    </div>
  );
}

const GRADIENTS = [
  "bg-gradient-to-br from-yellow-500 via-amber-600 to-orange-700",
  "bg-gradient-to-br from-pink-500 via-rose-600 to-purple-700",
  "bg-gradient-to-br from-yellow-400 via-yellow-600 to-amber-800",
  "bg-gradient-to-br from-pink-500 via-fuchsia-600 to-violet-700",
  "bg-gradient-to-br from-amber-500 via-yellow-600 to-yellow-800",
  "bg-gradient-to-br from-purple-500 via-pink-600 to-rose-700",
];

function gradientFor(id: string): string {
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0;
  return GRADIENTS[h % GRADIENTS.length];
}
