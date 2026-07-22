import { useEffect, useId, useRef, useState } from "react";
import { Loader2, MapPin, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { searchPlaces, type PlaceResult } from "@/lib/geocoding";

export interface LocationSelection {
  venueName: string;
  locationAddress: string;
  lat: number;
  lng: number;
}

interface LocationSearchProps {
  value: LocationSelection | null;
  onChange: (value: LocationSelection | null) => void;
  onVenueNameChange?: (name: string) => void;
  error?: string;
  disabled?: boolean;
}

export function LocationSearch({
  value,
  onChange,
  onVenueNameChange,
  error,
  disabled,
}: LocationSearchProps) {
  const listId = useId();
  const containerRef = useRef<HTMLDivElement>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<PlaceResult[]>([]);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [searchError, setSearchError] = useState<string | null>(null);

  useEffect(() => {
    if (!open || query.trim().length < 2) {
      setSuggestions([]);
      setLoading(false);
      return;
    }

    const controller = new AbortController();
    const timer = setTimeout(async () => {
      setLoading(true);
      setSearchError(null);
      try {
        const results = await searchPlaces(query, controller.signal);
        setSuggestions(results);
        setActiveIndex(results.length > 0 ? 0 : -1);
      } catch (err) {
        if (controller.signal.aborted) return;
        setSuggestions([]);
        setSearchError(err instanceof Error ? err.message : "Search failed");
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, 280);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [query, open]);

  useEffect(() => {
    function onPointerDown(e: MouseEvent) {
      if (!containerRef.current?.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, []);

  function selectPlace(place: PlaceResult) {
    onChange({
      venueName: place.venueName,
      locationAddress: place.locationAddress,
      lat: place.lat,
      lng: place.lng,
    });
    setQuery("");
    setOpen(false);
    setSuggestions([]);
  }

  function clearSelection() {
    onChange(null);
    setQuery("");
    setOpen(true);
  }

  if (value) {
    return (
      <div className="space-y-3">
        <div className="flex items-start gap-3 rounded-xl border border-border bg-card/50 px-4 py-3">
          <MapPin className="mt-0.5 size-4 shrink-0 text-primary" />
          <div className="min-w-0 flex-1">
            <Input
              value={value.venueName}
              onChange={(e) => {
                const next = { ...value, venueName: e.target.value };
                onChange(next);
                onVenueNameChange?.(e.target.value);
              }}
              disabled={disabled}
              className="h-auto border-0 bg-transparent p-0 text-sm font-medium shadow-none focus-visible:ring-0"
              aria-label="Venue name"
            />
            <p className="mt-0.5 text-sm text-muted-foreground">{value.locationAddress}</p>
          </div>
          <button
            type="button"
            onClick={clearSelection}
            disabled={disabled}
            className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label="Clear location"
          >
            <X className="size-4" />
          </button>
        </div>
        {error && <p className="text-sm text-destructive">{error}</p>}
      </div>
    );
  }

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <MapPin className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={(e) => {
            if (!open || suggestions.length === 0) return;
            if (e.key === "ArrowDown") {
              e.preventDefault();
              setActiveIndex((i) => (i + 1) % suggestions.length);
            } else if (e.key === "ArrowUp") {
              e.preventDefault();
              setActiveIndex((i) => (i <= 0 ? suggestions.length - 1 : i - 1));
            } else if (e.key === "Enter" && activeIndex >= 0) {
              e.preventDefault();
              selectPlace(suggestions[activeIndex]);
            } else if (e.key === "Escape") {
              setOpen(false);
            }
          }}
          placeholder="Search for an address or venue"
          disabled={disabled}
          role="combobox"
          aria-expanded={open}
          aria-controls={listId}
          aria-invalid={!!error}
          className="h-11 pl-9"
        />
        {loading && (
          <Loader2 className="absolute top-1/2 right-3 size-4 -translate-y-1/2 animate-spin text-muted-foreground" />
        )}
      </div>

      {open && (suggestions.length > 0 || searchError || (query.length >= 2 && !loading)) && (
        <ul
          id={listId}
          role="listbox"
          className="absolute z-50 mt-1 max-h-64 w-full overflow-auto rounded-xl border border-border bg-popover py-1 shadow-lg"
        >
          {searchError && (
            <li className="px-3 py-2 text-sm text-muted-foreground">{searchError}</li>
          )}
          {suggestions.map((place, i) => (
            <li key={`${place.lat}-${place.lng}-${i}`} role="option" aria-selected={i === activeIndex}>
              <button
                type="button"
                onMouseEnter={() => setActiveIndex(i)}
                onClick={() => selectPlace(place)}
                className={cn(
                  "flex w-full items-start gap-3 px-3 py-2.5 text-left transition-colors",
                  i === activeIndex ? "bg-muted" : "hover:bg-muted/60",
                )}
              >
                <MapPin className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium">{place.venueName}</span>
                  <span className="block truncate text-xs text-muted-foreground">
                    {place.locationAddress}
                  </span>
                </span>
              </button>
            </li>
          ))}
          {!loading && !searchError && suggestions.length === 0 && query.length >= 2 && (
            <li className="px-3 py-2 text-sm text-muted-foreground">No places found</li>
          )}
        </ul>
      )}

      {error && <p className="mt-1.5 text-sm text-destructive">{error}</p>}
    </div>
  );
}