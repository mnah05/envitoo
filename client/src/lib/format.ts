import { format, isSameDay, parseISO } from "date-fns";

/** Parse an ISO datetime string from the backend into a Date. */
export function parseDate(s: string): Date {
  return parseISO(s);
}

/** "Sat, Jun 26 · 7:00 PM" — Luma-style compact event date. */
export function formatEventDate(startsAt: string, endsAt?: string): string {
  const start = parseDate(startsAt);
  const day = format(start, "EEE, MMM d");
  const time = format(start, "h:mm a");
  let end = "";
  if (endsAt) {
    const e = parseDate(endsAt);
    end = isSameDay(start, e) ? ` – ${format(e, "h:mm a")}` : ` – ${format(e, "EEE, MMM d, h:mm a")}`;
  }
  return `${day} · ${time}${end}`;
}

/** Short date for cards: "Jun 26". */
export function formatCardDate(startsAt: string): string {
  return format(parseDate(startsAt), "MMM d");
}

/** Time range for cards: "7:00 PM". */
export function formatCardTime(startsAt: string): string {
  return format(parseDate(startsAt), "h:mm a");
}

/** Relative day label for landing/discover headers. */
export function formatFullDate(startsAt: string): string {
  return format(parseDate(startsAt), "EEEE, MMMM d, yyyy");
}

/**
 * Format a price stored as paise (1/100 of the major unit) with the
 * event's ISO 4217 currency code. Free events render as "Free".
 */
export function formatPrice(priceInPaise: number, currency: string): string {
  if (priceInPaise <= 0) return "Free";
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
      maximumFractionDigits: currency === "INR" ? 0 : 2,
    }).format(priceInPaise / 100);
  } catch {
    return `${(priceInPaise / 100).toFixed(2)} ${currency}`;
  }
}

/** First-letter avatar fallback for a name. */
export function initials(name: string): string {
  return name
    .split(" ")
    .map((p) => p[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}
