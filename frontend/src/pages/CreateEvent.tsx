import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  ArrowLeft,
  Calendar,
  Check,
  Copy,
  ExternalLink,
  Globe,
  MapPin,
  Sparkles,
  Ticket,
  Users,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { DateTimePicker } from "@/components/ui/date-time-picker";
import { Switch } from "@/components/ui/switch";
import { Spinner } from "@/components/ui/spinner";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PageContainer } from "@/components/domain/Layout";
import { useCreateEvent } from "@/hooks/use-events";
import { CreateEventInput, CATEGORIES, CATEGORY_LABELS, type CreateEventValues } from "@/lib/api";
import {
  LocationSearch,
  type LocationSelection,
} from "@/components/domain/LocationSearch";
import { cn } from "@/lib/utils";

interface FormState {
  title: string;
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

function defaultDateTime(offsetHours = 1): string {
  const d = new Date();
  d.setHours(d.getHours() + offsetHours, 0, 0, 0);
  return d.toISOString().slice(0, 16);
}

function createInitialState(): FormState {
  const starts_at = defaultDateTime(1);
  const ends_at = defaultDateTime(3);
  return {
    title: "",
    description: "",
    starts_at,
    ends_at,
    capacity: "50",
    requires_approval: false,
    visibility: "public",
    cover_image_url: "",
    is_virtual: false,
    meeting_url: "",
    category: "",
  };
}

export function CreateEvent() {
  const navigate = useNavigate();
  const createMut = useCreateEvent();
  const [form, setForm] = useState<FormState>(createInitialState);
  const [location, setLocation] = useState<LocationSelection | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [createdEvent, setCreatedEvent] = useState<{ id: string; slug: string; title: string } | null>(null);

  const eventUrl = createdEvent
    ? `${window.location.origin}/e/${createdEvent.slug}`
    : "";

  const [copied, setCopied] = useState(false);

  const copyLink = useCallback(async () => {
    if (!eventUrl) return;
    try {
      await navigator.clipboard.writeText(eventUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error("Could not copy link");
    }
  }, [eventUrl]);

  useEffect(() => {
    document.title = "Create event — EnviToo";
  }, []);

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  function buildPayload(): CreateEventValues | { errors: Record<string, string> } {
    const errors: Record<string, string> = {};

    if (!form.title.trim()) errors.title = "Give your event a name";
    if (!form.starts_at) errors.starts_at = "Pick a start time";
    if (!form.ends_at) errors.ends_at = "Pick an end time";
    else if (form.starts_at && new Date(form.ends_at) <= new Date(form.starts_at))
      errors.ends_at = "End time must be after start";
    const cap = Number(form.capacity);
    if (!form.capacity || !Number.isFinite(cap) || cap < 1)
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

    const payload: CreateEventValues = {
      title: form.title.trim(),
      description: form.description.trim() || undefined,
      starts_at: new Date(form.starts_at).toISOString(),
      ends_at: new Date(form.ends_at).toISOString(),
      capacity: Math.round(cap),
      price_in_paise: 0,
      currency: "INR",
      requires_approval: form.requires_approval,
      cover_image_url: form.cover_image_url.trim() || undefined,
      is_virtual: form.is_virtual,
      meeting_url: form.is_virtual ? form.meeting_url.trim() : undefined,
      venue_name: !form.is_virtual && location ? location.venueName.trim() : undefined,
      location_address:
        !form.is_virtual && location ? location.locationAddress.trim() : undefined,
      lat: !form.is_virtual && location ? location.lat : undefined,
      lng: !form.is_virtual && location ? location.lng : undefined,
      category: (form.category || undefined) as CreateEventValues["category"],
      visibility: form.visibility === "unlisted" ? "unlisted" : undefined,
    };

    const parsed = CreateEventInput.safeParse(payload);
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
    if (createMut.isPending) return;
    const result = buildPayload();
    if ("errors" in result) {
      setFieldErrors(result.errors);
      return;
    }
    setFieldErrors({});
    try {
      const event = await createMut.mutateAsync(result);
      setCreatedEvent({ id: event.id, slug: event.slug, title: event.title });
    } catch {
      // mutation onError surfaces a toast
    }
  }

  return (
    <div className="relative min-h-screen pt-16">
      <PageContainer width="narrow" className="py-10 sm:py-14">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mb-6 -ml-2 text-muted-foreground hover:text-foreground"
          onClick={() => navigate(-1)}
        >
          <ArrowLeft data-icon="inline-start" />
          Back
        </Button>

        <div className="mb-10">
          <h1 className="display-title text-3xl sm:text-4xl">Create event</h1>
          <p className="text-lede mt-3 text-[1.0625rem]">
            Set the basics — you can fine-tune details later.
          </p>
        </div>

        <form onSubmit={onSubmit} className="space-y-8">
          {/* Title */}
          <div>
            <label htmlFor="title" className="label-muted mb-2 block">
              Event name
            </label>
            <input
              id="title"
              value={form.title}
              onChange={(e) => set("title", e.target.value)}
              placeholder="What's happening?"
              aria-invalid={!!fieldErrors.title}
              className={cn(
                "font-display w-full border-0 bg-transparent text-2xl font-semibold tracking-[-0.028em] outline-none placeholder:text-muted-foreground/40 sm:text-3xl",
                fieldErrors.title && "text-destructive placeholder:text-destructive/40",
              )}
            />
            {fieldErrors.title && (
              <p className="mt-1 text-sm text-destructive">{fieldErrors.title}</p>
            )}
          </div>

          {/* Meta rows */}
          <div className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
            {/* Date & time */}
            <div className="flex items-start gap-4 px-4 py-4">
              <Calendar className="mt-2.5 size-5 shrink-0 text-primary/80" />
              <div className="grid flex-1 gap-3 sm:grid-cols-2">
                <div>
<label htmlFor="starts_at" className="label-muted mb-1 block">
                      Start
                    </label>
                  <DateTimePicker
                    id="starts_at"
                    value={form.starts_at}
                    onChange={(v) => set("starts_at", v)}
                    aria-invalid={!!fieldErrors.starts_at}
                    placeholder="Pick start date & time"
                  />
                  {fieldErrors.starts_at && (
                    <p className="text-xs text-destructive">{fieldErrors.starts_at}</p>
                  )}
                </div>
                <div>
<label htmlFor="ends_at" className="label-muted mb-1 block">
                      End
                    </label>
                  <DateTimePicker
                    id="ends_at"
                    value={form.ends_at}
                    onChange={(v) => set("ends_at", v)}
                    aria-invalid={!!fieldErrors.ends_at}
                    placeholder="Pick end date & time"
                  />
                  {fieldErrors.ends_at && (
                    <p className="text-xs text-destructive">{fieldErrors.ends_at}</p>
                  )}
                </div>
              </div>
            </div>

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
                <div className="flex rounded-lg border border-border bg-muted/40 p-0.5 text-xs">
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
                      "min-h-[36px] rounded-md px-3.5 py-1.5 font-medium transition-colors",
                      !form.is_virtual
                        ? "bg-primary text-primary-foreground shadow-sm shadow-primary/20"
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
                      "min-h-[36px] rounded-md px-3.5 py-1.5 font-medium transition-colors",
                      form.is_virtual
                        ? "bg-primary text-primary-foreground shadow-sm shadow-primary/20"
                        : "text-muted-foreground hover:text-foreground",
                    )}
                  >
                    Online
                  </button>
                </div>
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

            {/* Free ticket */}
            <div className="flex items-center gap-4 px-4 py-4">
              <Ticket className="size-5 shrink-0 text-primary/80" />
              <div className="flex flex-1 items-center justify-between gap-3">
                <div>
                  <p className="text-sm font-medium">Free event</p>
                  <p className="text-xs text-muted-foreground">Paid tickets coming soon</p>
                </div>
                <Badge variant="secondary" className="shrink-0 border-primary/20 bg-primary/10 text-primary">
                  Free
                </Badge>
              </div>
            </div>

            {/* Capacity */}
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

            {/* Require approval */}
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

            {/* Visibility */}
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
          </div>

          {/* Category */}
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
                disabled={createMut.isPending}
                className="ml-auto rounded-full px-6 sm:min-w-[10rem]"
              >
                {createMut.isPending && <Spinner data-icon="inline-start" />}
                Create event
              </Button>
            </PageContainer>
          </div>
        </form>
      </PageContainer>

      {/* Post-creation share dialog */}
      <Dialog open={!!createdEvent} onOpenChange={(open) => {
        if (!open && createdEvent) {
          navigate(`/manage/${createdEvent.id}`);
        }
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-xl">
              <span className="grid size-8 place-items-center rounded-full bg-primary/20 text-primary">
                <Sparkles className="size-4" />
              </span>
              Event published!
            </DialogTitle>
            <DialogDescription className="pt-1 text-base">
              Your event <span className="display-headline text-foreground">{createdEvent?.title}</span> is now live.
            </DialogDescription>
          </DialogHeader>

          {eventUrl && (
            <div className="space-y-3">
              <label className="label-muted">
                Shareable link
              </label>
              <div className="flex items-center gap-2 rounded-xl border border-border bg-muted/50 px-3 py-2.5">
                <ExternalLink className="size-4 shrink-0 text-primary" />
                <span className="min-w-0 truncate text-sm text-muted-foreground">
                  {eventUrl}
                </span>
              </div>
              <div className="flex gap-2">
                <Button onClick={copyLink} className="flex-1 gap-2">
                  {copied ? (
                    <><Check className="size-4" /> Copied!</>
                  ) : (
                    <><Copy className="size-4" /> Copy link</>
                  )}
                </Button>
                <Button
                  variant="outline"
                  className="flex-1 gap-2"
                  onClick={() => navigate(`/e/${createdEvent!.slug}`, { replace: true })}
                >
                  <ExternalLink className="size-4" />
                  View page
                </Button>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="w-full text-muted-foreground"
                onClick={() => navigate(`/manage/${createdEvent!.id}`, { replace: true })}
              >
                Go to dashboard
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}