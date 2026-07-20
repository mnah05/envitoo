import { useState, useEffect } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  ArrowLeft,
  Calendar,
  Globe,
  MapPin,
  ShieldAlert,
  Ticket,
  Users,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Spinner } from "@/components/ui/spinner";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { PageContainer } from "@/components/domain/Layout";
import { Skeleton } from "@/components/ui/skeleton";
import {
  LocationSearch,
  type LocationSelection,
} from "@/components/domain/LocationSearch";
import { useEvent, useUpdateEvent } from "@/hooks/use-events";
import { useMe } from "@/hooks/use-me";
import { useMyEventStaff } from "@/hooks/use-volunteers";
import {
  CATEGORIES,
  CATEGORY_LABELS,
  UpdateEventInput,
  type UpdateEventValues,
} from "@/lib/api";
import { cn } from "@/lib/utils";

function isoToLocalInput(iso: string): string {
  const d = new Date(iso);
  const off = d.getTimezoneOffset();
  const local = new Date(d.getTime() - off * 60_000);
  return local.toISOString().slice(0, 16);
}

interface FormState {
  description: string;
  starts_at: string;
  ends_at: string;
  capacity: string;
  requires_approval: boolean;
  visibility: "public" | "unlisted";
  cover_image_url: string;
  is_virtual: boolean;
  meeting_url: string;
  category: string;
}

export function EditEvent() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: event, isLoading: eventLoading } = useEvent(id, !!id);
  const { data: me } = useMe();
  const { data: staffData } = useMyEventStaff({}, !!id);

  const updateMut = useUpdateEvent();
  const [form, setForm] = useState<FormState | null>(null);
  const [location, setLocation] = useState<LocationSelection | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const isOrganizer = !!event && !!me && event.organizer_id === me.id;
  const isVolunteerAdmin =
    !isOrganizer && staffData?.events.some((e) => e.event_id === id) === true;

  useEffect(() => {
    document.title = event ? `Edit ${event.title} — EnviToo` : "Edit — EnviToo";
  }, [event]);

  useEffect(() => {
    if (event && !form) {
      setForm({
        description: event.description ?? "",
        starts_at: isoToLocalInput(event.starts_at),
        ends_at: isoToLocalInput(event.ends_at),
        capacity: String(event.capacity),
        requires_approval: event.requires_approval,
        visibility: event.visibility ?? "public",
        cover_image_url: event.cover_image_url ?? "",
        is_virtual: event.is_virtual,
        meeting_url: event.meeting_url ?? "",
        category: event.category ?? "",
      });

      if (!event.is_virtual && event.lat != null && event.lng != null) {
        setLocation({
          venueName: event.venue_name ?? "",
          locationAddress: event.location_address ?? event.venue_name ?? "",
          lat: event.lat,
          lng: event.lng,
        });
      }
    }
  }, [event, form]);

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((f) => (f ? { ...f, [key]: value } : f));
  }

  function buildPayload(): UpdateEventValues | { errors: Record<string, string> } {
    if (!form) return { errors: {} };
    const errors: Record<string, string> = {};

    if (isOrganizer && form.starts_at) {
      if (!form.ends_at || new Date(form.ends_at) <= new Date(form.starts_at))
        errors.ends_at = "End time must be after start";
    }
    const cap = Number(form.capacity);
    if (isOrganizer && form.capacity && (!Number.isFinite(cap) || cap < 1))
      errors.capacity = "Capacity must be at least 1";

    if (form.is_virtual) {
      if (!form.meeting_url.trim()) errors.meeting_url = "Add a meeting link";
      else if (!/^https?:\/\//i.test(form.meeting_url)) errors.meeting_url = "Enter a valid URL";
    } else if (!location) {
      errors.location = "Search and select a location";
    } else {
      if (!location.venueName.trim()) errors.location = "Venue name is required";
      if (!location.locationAddress.trim()) errors.location = "Address is required";
    }

    if (form.cover_image_url && !/^https?:\/\//i.test(form.cover_image_url))
      errors.cover_image_url = "Enter a valid image URL";

    if (Object.keys(errors).length > 0) return { errors };

    const payload: UpdateEventValues = {
      description: form.description.trim() || undefined,
      cover_image_url: form.cover_image_url.trim() || undefined,
      meeting_url: form.is_virtual ? form.meeting_url.trim() : undefined,
      venue_name: !form.is_virtual && location ? location.venueName.trim() : undefined,
      location_address:
        !form.is_virtual && location ? location.locationAddress.trim() : undefined,
      lat: !form.is_virtual && location ? location.lat : undefined,
      lng: !form.is_virtual && location ? location.lng : undefined,
      is_virtual: form.is_virtual,
    };

    if (isOrganizer) {
      if (form.starts_at) payload.starts_at = new Date(form.starts_at).toISOString();
      if (form.ends_at) payload.ends_at = new Date(form.ends_at).toISOString();
      if (form.capacity) payload.capacity = Math.round(cap);
      payload.requires_approval = form.requires_approval;
      payload.category = (form.category || undefined) as UpdateEventValues["category"];
      if (form.visibility !== event.visibility) {
        payload.visibility = form.visibility;
      }
    }

    const parsed = UpdateEventInput.safeParse(payload);
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        errors[issue.path[0] as string] = errors[issue.path[0] as string] ?? issue.message;
      }
      return { errors };
    }
    return parsed.data;
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (updateMut.isPending) return;
    const result = buildPayload();
    if ("errors" in result) {
      setFieldErrors(result.errors);
      return;
    }
    setFieldErrors({});
    try {
      await updateMut.mutateAsync({ id, values: result });
      navigate(`/manage/${id}`);
    } catch {
      // mutation onError surfaces a toast
    }
  }

  if (eventLoading && !event) {
    return (
      <div className="pt-16">
        <PageContainer width="narrow" className="py-10">
          <Skeleton className="mb-4 h-8 w-24" />
          <Skeleton className="h-9 w-48" />
          <Skeleton className="mt-2 h-5 w-64" />
          <div className="mt-8 flex flex-col gap-5">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full rounded-xl" />
            ))}
          </div>
        </PageContainer>
      </div>
    );
  }

  if (!event || !form) {
    return (
      <div className="pt-16">
        <PageContainer className="flex min-h-[50vh] flex-col items-center justify-center gap-4 text-center">
          <h1 className="display-title text-2xl">Event not found</h1>
          <p className="text-lede text-[1.0625rem]">You may not have access to this event.</p>
          <Button asChild variant="outline">
            <Link to="/manage">Back to my events</Link>
          </Button>
        </PageContainer>
      </div>
    );
  }

  const isFree = event.price_in_paise === 0;

  return (
    <div className="relative min-h-screen pt-16">
      <PageContainer width="narrow" className="py-10 sm:py-14">
        <Button
          asChild
          variant="ghost"
          size="sm"
          className="mb-6 -ml-2 text-muted-foreground hover:text-foreground"
        >
          <Link to={`/manage/${id}`}>
            <ArrowLeft data-icon="inline-start" />
            Back to dashboard
          </Link>
        </Button>

        <div className="mb-10">
          <h1 className="display-title text-3xl sm:text-4xl">Update event</h1>
          <p className="text-lede mt-3 text-[1.0625rem]">
            Fine-tune details, location, and settings.
          </p>
        </div>

        {isVolunteerAdmin && (
          <Alert className="mb-8 border-primary/20 bg-card/60">
            <ShieldAlert className="size-4 text-primary" />
            <AlertTitle>Volunteer admin access</AlertTitle>
            <AlertDescription>
              You can edit description, cover image, and location. Scheduling, capacity, and ticket
              settings require the organizer.
            </AlertDescription>
          </Alert>
        )}

        <form onSubmit={onSubmit} className="space-y-8">
          {/* Event title (read-only) */}
          <div>
            <p className="label-muted mb-2">
              Event name
            </p>
            <p className="display-title text-2xl sm:text-3xl">{event.title}</p>
          </div>

          {/* Meta card */}
          <div className="divide-y divide-border/60 overflow-hidden rounded-xl border border-self-strong bg-card">
            {isOrganizer && (
              <div className="flex items-start gap-4 px-4 py-4">
                <Calendar className="mt-2.5 size-5 shrink-0 text-primary/80" />
                <div className="grid flex-1 gap-3 sm:grid-cols-2">
                  <div>
                    <label htmlFor="starts_at" className="label-muted mb-1 block">
                      Start
                    </label>
                    <Input
                      id="starts_at"
                      type="datetime-local"
                      value={form.starts_at}
                      onChange={(e) => set("starts_at", e.target.value)}
                      aria-invalid={!!fieldErrors.starts_at}
                      className="h-10 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0"
                    />
                  </div>
                  <div>
                    <label htmlFor="ends_at" className="label-muted mb-1 block">
                      End
                    </label>
                    <Input
                      id="ends_at"
                      type="datetime-local"
                      value={form.ends_at}
                      onChange={(e) => set("ends_at", e.target.value)}
                      aria-invalid={!!fieldErrors.ends_at}
                      className="h-10 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0"
                    />
                    {fieldErrors.ends_at && (
                      <p className="text-xs text-destructive">{fieldErrors.ends_at}</p>
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* Location */}
            <div className="px-4 py-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  {form.is_virtual ? (
                    <Globe className="size-5 text-primary/80" />
                  ) : (
                    <MapPin className="size-5 text-primary/80" />
                  )}
                  <span className="text-sm font-medium">
                    {form.is_virtual ? "Online event" : "In-person"}
                  </span>
                </div>
                {isOrganizer && (
                  <div className="flex rounded-lg border border-border/50 bg-muted/30 p-0.5 text-xs">
                    <button
                      type="button"
                      onClick={() => {
                        set("is_virtual", false);
                        setFieldErrors((e) => {
                          const next = { ...e };
                          delete next.meeting_url;
                          return next;
                        });
                      }}
                      className={cn(
                        "min-h-[34px] rounded-md px-3.5 py-1.5 font-medium transition-colors",
                        !form.is_virtual
                          ? "bg-primary text-primary-foreground"
                          : "text-muted-foreground hover:text-foreground",
                      )}
                    >
                      In person
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        set("is_virtual", true);
                        setLocation(null);
                        setFieldErrors((e) => {
                          const next = { ...e };
                          delete next.location;
                          return next;
                        });
                      }}
                      className={cn(
                        "min-h-[34px] rounded-md px-3.5 py-1.5 font-medium transition-colors",
                        form.is_virtual
                          ? "bg-primary text-primary-foreground"
                          : "text-muted-foreground hover:text-foreground",
                      )}
                    >
                      Online
                    </button>
                  </div>
                )}
              </div>

              {form.is_virtual ? (
                <Input
                  type="url"
                  value={form.meeting_url}
                  onChange={(e) => set("meeting_url", e.target.value)}
                  placeholder="https://zoom.us/j/..."
                  aria-invalid={!!fieldErrors.meeting_url}
                  className="h-11"
                />
              ) : (
                <LocationSearch
                  value={location}
                  onChange={setLocation}
                  error={fieldErrors.location}
                />
              )}
              {fieldErrors.meeting_url && (
                <p className="mt-1.5 text-sm text-destructive">{fieldErrors.meeting_url}</p>
              )}
            </div>

            {isOrganizer && (
              <>
                <div className="flex items-center gap-4 px-4 py-4">
                  <Ticket className="size-5 shrink-0 text-primary/80" />
                  <div className="flex flex-1 items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">
                        {isFree ? "Free event" : `Paid · ${event.currency}`}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {isFree ? "Paid tickets coming soon" : "Pricing changes coming soon"}
                      </p>
                    </div>
                    {isFree ? (
                      <Badge variant="secondary" className="border-primary/20 bg-primary/10 text-primary">
                        Free
                      </Badge>
                    ) : (
                      <Badge variant="outline">
                        {(event.price_in_paise / 100).toFixed(2)} {event.currency}
                      </Badge>
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-4 px-4 py-4">
                  <Users className="size-5 shrink-0 text-primary/80" />
                  <div className="flex flex-1 items-center justify-between gap-3">
                    <label htmlFor="capacity" className="text-sm font-medium">
                      Capacity
                    </label>
                    <Input
                      id="capacity"
                      type="number"
                      min={1}
                      value={form.capacity}
                      onChange={(e) => set("capacity", e.target.value)}
                      aria-invalid={!!fieldErrors.capacity}
                      className="h-9 w-24 text-right"
                    />
                  </div>
                  {fieldErrors.capacity && (
                    <p className="text-xs text-destructive">{fieldErrors.capacity}</p>
                  )}
                </div>
              </>
            )}

          {/* Description */}
            <div className="px-4 py-4">
              <label htmlFor="description" className="label-muted mb-2 block">
                Description
              </label>
              <Textarea
                id="description"
                value={form.description}
                onChange={(e) => set("description", e.target.value)}
                placeholder="Tell people what to expect…"
                rows={5}
                className="resize-none rounded-xl border-border bg-background text-base shadow-none focus-visible:border-primary/40 focus-visible:ring-primary/20"
              />
            </div>

            {/* Cover image URL */}
            <div className="px-4 py-4">
              <label htmlFor="cover_image_url" className="label-muted mb-1.5 block">
                Cover image URL <span className="text-muted-foreground/70">(optional)</span>
              </label>
              <Input
                id="cover_image_url"
                type="url"
                value={form.cover_image_url}
                onChange={(e) => set("cover_image_url", e.target.value)}
                placeholder="https://…"
                aria-invalid={!!fieldErrors.cover_image_url}
                className="h-10"
              />
              {fieldErrors.cover_image_url && (
                <p className="mt-1 text-xs text-destructive">{fieldErrors.cover_image_url}</p>
              )}
            </div>

            {isOrganizer && (
              <div className="flex items-center justify-between gap-4 px-4 py-4">
                <div>
                  <p className="text-sm font-medium">Require approval to register</p>
                  <p className="text-xs text-muted-foreground">Review guests before they join</p>
                </div>
                <Switch
                  checked={form.requires_approval}
                  onCheckedChange={(v) => set("requires_approval", v)}
                />
              </div>
            )}

            {isOrganizer && (
              <div className="flex items-center justify-between gap-4 px-4 py-4">
                <div>
                  <p className="text-sm font-medium">List on Discover page</p>
                  <p className="text-xs text-muted-foreground">
                    Unlisted events are only accessible via direct link
                  </p>
                </div>
                <Switch
                  checked={form.visibility === "public"}
                  onCheckedChange={(v) => set("visibility", v ? "public" : "unlisted")}
                />
              </div>
            )}
          </div>

          {/* Category — organizer only */}
          {isOrganizer && (
            <div>
              <p className="label-muted mb-3">Category</p>
              <div className="flex flex-wrap gap-2">
                {CATEGORIES.map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => set("category", form.category === c ? "" : c)}
                    className={cn(
                      "rounded-full border px-3.5 py-1.5 text-sm font-medium transition-colors",
                      form.category === c
                        ? "border-primary/50 bg-primary/15 text-primary"
                        : "border-border text-muted-foreground hover:border-primary/30 hover:text-foreground",
                    )}
                  >
                    {CATEGORY_LABELS[c]}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Spacer so sticky footer / bottom nav doesn't cover content */}
          <div className="h-32 sm:h-24" aria-hidden />

          {/* Sticky action footer (inside form so submit associates natively) */}
          <div className="fixed inset-x-0 bottom-16 z-30 border-t bg-background/90 backdrop-blur-md sm:bottom-0 sm:pb-[env(safe-area-inset-bottom)]">
            <PageContainer width="narrow" className="flex items-center justify-between gap-3 py-3">
              <Button
                type="button"
                variant="ghost"
                onClick={() => navigate(-1)}
                className="text-muted-foreground hover:text-foreground sm:hidden"
              >
                Cancel
              </Button>
            <Button
              type="submit"
              size="sm"
              disabled={updateMut.isPending}
              className="ml-auto px-6 sm:min-w-[10rem]"
            >
              {updateMut.isPending && <Spinner data-icon="inline-start" />}
              Save changes
            </Button>
            </PageContainer>
          </div>
        </form>
      </PageContainer>
    </div>
  );
}