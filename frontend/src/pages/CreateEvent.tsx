import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useEffect } from "react";
import { LocateFixed, Calendar, MapPin, Globe, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Spinner } from "@/components/ui/spinner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldSet,
} from "@/components/ui/field";
import { useCreateEvent } from "@/hooks/use-events";
import { CreateEventInput, CATEGORIES, CATEGORY_LABELS, type CreateEventValues } from "@/lib/api";

const CURRENCIES = ["INR", "USD", "EUR", "GBP", "AED", "SGD", "AUD", "CAD", "JPY"];

interface FormState {
  title: string;
  description: string;
  starts_at: string;
  ends_at: string;
  capacity: string;
  price: string;
  currency: string;
  requires_approval: boolean;
  cover_image_url: string;
  is_virtual: string;
  meeting_url: string;
  venue_name: string;
  lat: string;
  lng: string;
  category: string;
}

const initial: FormState = {
  title: "",
  description: "",
  starts_at: "",
  ends_at: "",
  capacity: "50",
  price: "0",
  currency: "INR",
  requires_approval: false,
  cover_image_url: "",
  is_virtual: "physical",
  meeting_url: "",
  venue_name: "",
  lat: "",
  lng: "",
  category: "",
};

export function CreateEvent() {
  const navigate = useNavigate();
  const createMut = useCreateEvent();
  const [form, setForm] = useState<FormState>(initial);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [isFree, setIsFree] = useState(true);

  useEffect(() => {
    document.title = "Create event — EnviToo";
  }, []);

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  function useMyLocation() {
    if (!navigator.geolocation) return;
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        set("lat", pos.coords.latitude.toFixed(6));
        set("lng", pos.coords.longitude.toFixed(6));
      },
      () => undefined,
    );
  }

  function buildPayload(): CreateEventValues | { errors: Record<string, string> } {
    const isVirtual = form.is_virtual === "virtual";
    const errors: Record<string, string> = {};

    if (!form.title.trim()) errors.title = "Title is required";
    if (!form.starts_at) errors.starts_at = "Start time is required";
    if (!form.ends_at) errors.ends_at = "End time is required";
    else if (form.starts_at && new Date(form.ends_at) <= new Date(form.starts_at))
      errors.ends_at = "End time must be after the start time";
    const cap = Number(form.capacity);
    if (!form.capacity || !Number.isFinite(cap) || cap < 1) errors.capacity = "Capacity must be at least 1";
    const price = Number(form.price);
    if (!Number.isFinite(price) || price < 0) errors.price = "Price must be 0 or more";

    if (isVirtual) {
      if (!form.meeting_url.trim()) errors.meeting_url = "Meeting URL is required for virtual events";
      else if (!/^https?:\/\//i.test(form.meeting_url)) errors.meeting_url = "Enter a valid URL";
    } else {
      if (!form.venue_name.trim()) errors.venue_name = "Venue is required for in-person events";
      const lat = Number(form.lat);
      const lng = Number(form.lng);
      if (!Number.isFinite(lat) || lat < -90 || lat > 90) errors.lat = "Enter a valid latitude";
      if (!Number.isFinite(lng) || lng < -180 || lng > 180) errors.lng = "Enter a valid longitude";
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
      price_in_paise: Math.round(price * 100),
      currency: form.currency,
      requires_approval: form.requires_approval,
      cover_image_url: form.cover_image_url.trim() || undefined,
      is_virtual: isVirtual,
      meeting_url: isVirtual ? form.meeting_url.trim() : undefined,
      venue_name: !isVirtual ? form.venue_name.trim() : undefined,
      lat: !isVirtual ? Number(form.lat) : undefined,
      lng: !isVirtual ? Number(form.lng) : undefined,
      category: (form.category || undefined) as CreateEventValues["category"],
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
      navigate(`/manage/${event.id}`);
    } catch {
      // mutation onError surfaces a toast
    }
  }

  const isVirtual = form.is_virtual === "virtual";

  return (
    <div className="min-h-screen pt-16">
      <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
        {/* Header */}
        <div className="mb-10">
          <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Create an event
          </h1>
          <p className="mt-2 text-muted-foreground">
            Set up your event page, invite friends, and sell tickets.
          </p>
        </div>

        <form onSubmit={onSubmit}>
          <div className="grid gap-8 lg:grid-cols-[1fr_280px]">
            {/* ─── Main form ─── */}
            <div className="space-y-10">
              {/* Basic info */}
              <section>
                <div className="mb-6 flex items-center gap-3 border-b border-border pb-4">
                  <Sparkles className="size-5 text-primary" />
                  <h2 className="text-lg font-semibold">Basic info</h2>
                </div>
                <FieldGroup>
                  <Field data-invalid={!!fieldErrors.title}>
                    <FieldLabel htmlFor="title">Event title</FieldLabel>
                    <Input
                      id="title"
                      value={form.title}
                      onChange={(e) => set("title", e.target.value)}
                      placeholder="e.g. AI Builders Meetup"
                      aria-invalid={!!fieldErrors.title}
                      className="h-10"
                    />
                    {fieldErrors.title && <FieldError>{fieldErrors.title}</FieldError>}
                  </Field>

                  <Field>
                    <FieldLabel htmlFor="description">Description</FieldLabel>
                    <Textarea
                      id="description"
                      value={form.description}
                      onChange={(e) => set("description", e.target.value)}
                      placeholder="Tell people what your event is about"
                      rows={4}
                    />
                    <FieldDescription>Plain text. Up to 5000 characters.</FieldDescription>
                  </Field>

                  <Field data-invalid={!!fieldErrors.category}>
                    <FieldLabel htmlFor="category">Category</FieldLabel>
                    <Select value={form.category} onValueChange={(v) => set("category", v)}>
                      <SelectTrigger id="category" className="h-10">
                        <SelectValue placeholder="Choose a category" />
                      </SelectTrigger>
                      <SelectContent>
                        {CATEGORIES.map((c) => (
                          <SelectItem key={c} value={c}>
                            {CATEGORY_LABELS[c]}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                </FieldGroup>
              </section>

              {/* Date & time */}
              <section>
                <div className="mb-6 flex items-center gap-3 border-b border-border pb-4">
                  <Calendar className="size-5 text-primary" />
                  <h2 className="text-lg font-semibold">Date &amp; time</h2>
                </div>
                <FieldSet>
                  <div className="grid gap-5 sm:grid-cols-2">
                    <Field data-invalid={!!fieldErrors.starts_at}>
                      <FieldLabel htmlFor="starts_at">Starts</FieldLabel>
                      <Input
                        id="starts_at"
                        type="datetime-local"
                        value={form.starts_at}
                        onChange={(e) => set("starts_at", e.target.value)}
                        aria-invalid={!!fieldErrors.starts_at}
                        className="h-10"
                      />
                      {fieldErrors.starts_at && <FieldError>{fieldErrors.starts_at}</FieldError>}
                    </Field>
                    <Field data-invalid={!!fieldErrors.ends_at}>
                      <FieldLabel htmlFor="ends_at">Ends</FieldLabel>
                      <Input
                        id="ends_at"
                        type="datetime-local"
                        value={form.ends_at}
                        onChange={(e) => set("ends_at", e.target.value)}
                        aria-invalid={!!fieldErrors.ends_at}
                        className="h-10"
                      />
                      {fieldErrors.ends_at && <FieldError>{fieldErrors.ends_at}</FieldError>}
                    </Field>
                  </div>
                </FieldSet>
              </section>

              {/* Location */}
              <section>
                <div className="mb-6 flex items-center gap-3 border-b border-border pb-4">
                  <MapPin className="size-5 text-primary" />
                  <h2 className="text-lg font-semibold">Location</h2>
                </div>
                <FieldSet>
                  <Field>
                    <FieldLabel>Event format</FieldLabel>
                    <ToggleGroup
                      type="single"
                      value={form.is_virtual}
                      onValueChange={(v) => v && set("is_virtual", v)}
                      variant="outline"
                      className="w-full"
                    >
                      <ToggleGroupItem value="physical" className="flex-1 gap-2">
                        <MapPin className="size-4" />
                        In person
                      </ToggleGroupItem>
                      <ToggleGroupItem value="virtual" className="flex-1 gap-2">
                        <Globe className="size-4" />
                        Virtual
                      </ToggleGroupItem>
                    </ToggleGroup>
                  </Field>

                  {isVirtual ? (
                    <Field data-invalid={!!fieldErrors.meeting_url}>
                      <FieldLabel htmlFor="meeting_url">Meeting URL</FieldLabel>
                      <Input
                        id="meeting_url"
                        type="url"
                        value={form.meeting_url}
                        onChange={(e) => set("meeting_url", e.target.value)}
                        placeholder="https://meet.example/abc"
                        aria-invalid={!!fieldErrors.meeting_url}
                        className="h-10"
                      />
                      {fieldErrors.meeting_url && <FieldError>{fieldErrors.meeting_url}</FieldError>}
                    </Field>
                  ) : (
                    <div className="grid gap-5 sm:grid-cols-2">
                      <Field data-invalid={!!fieldErrors.venue_name} className="sm:col-span-2">
                        <FieldLabel htmlFor="venue_name">Venue name</FieldLabel>
                        <Input
                          id="venue_name"
                          value={form.venue_name}
                          onChange={(e) => set("venue_name", e.target.value)}
                          placeholder="e.g. Convention Center, Hall B"
                          aria-invalid={!!fieldErrors.venue_name}
                          className="h-10"
                        />
                        {fieldErrors.venue_name && <FieldError>{fieldErrors.venue_name}</FieldError>}
                      </Field>
                      <Field data-invalid={!!fieldErrors.lat}>
                        <FieldLabel htmlFor="lat">Latitude</FieldLabel>
                        <Input
                          id="lat"
                          type="number"
                          step="any"
                          value={form.lat}
                          onChange={(e) => set("lat", e.target.value)}
                          placeholder="12.9716"
                          aria-invalid={!!fieldErrors.lat}
                          className="h-10"
                        />
                        {fieldErrors.lat && <FieldError>{fieldErrors.lat}</FieldError>}
                      </Field>
                      <Field data-invalid={!!fieldErrors.lng}>
                        <FieldLabel htmlFor="lng">Longitude</FieldLabel>
                        <Input
                          id="lng"
                          type="number"
                          step="any"
                          value={form.lng}
                          onChange={(e) => set("lng", e.target.value)}
                          placeholder="77.5946"
                          aria-invalid={!!fieldErrors.lng}
                          className="h-10"
                        />
                        {fieldErrors.lng && <FieldError>{fieldErrors.lng}</FieldError>}
                      </Field>
                      <div className="sm:col-span-2">
                        <Button type="button" variant="outline" size="sm" onClick={useMyLocation}>
                          <LocateFixed data-icon="inline-start" />
                          Use my location
                        </Button>
                      </div>
                    </div>
                  )}
                </FieldSet>
              </section>

              {/* Cover image */}
              <section>
                <div className="mb-6 flex items-center gap-3 border-b border-border pb-4">
                  <Globe className="size-5 text-primary" />
                  <h2 className="text-lg font-semibold">Cover image</h2>
                </div>
                <Field data-invalid={!!fieldErrors.cover_image_url}>
                  <FieldLabel htmlFor="cover_image_url">Cover image URL</FieldLabel>
                  <Input
                    id="cover_image_url"
                    type="url"
                    value={form.cover_image_url}
                    onChange={(e) => set("cover_image_url", e.target.value)}
                    placeholder="https://…/cover.jpg"
                    aria-invalid={!!fieldErrors.cover_image_url}
                    className="h-10"
                  />
                  {fieldErrors.cover_image_url ? (
                    <FieldError>{fieldErrors.cover_image_url}</FieldError>
                  ) : (
                    <FieldDescription>Optional. A 16:9 image works best.</FieldDescription>
                  )}
                </Field>
              </section>
            </div>

            {/* ─── Sidebar — quick config ─── */}
            <div className="order-first lg:order-last">
              <div className="sticky top-24 space-y-6 rounded-2xl border border-border bg-card p-6">
                <div>
                  <h3 className="mb-4 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                    Tickets
                  </h3>
                  <FieldSet>
                    <Field>
                      <FieldLabel>Ticketed event</FieldLabel>
                      <Switch checked={!isFree} onCheckedChange={(v) => { setIsFree(!v); if (v) set("price", "0"); }} />
                    </Field>

                    {!isFree && (
                      <div className="grid grid-cols-2 gap-3">
                        <Field data-invalid={!!fieldErrors.price}>
                          <FieldLabel htmlFor="price">Price</FieldLabel>
                          <Input
                            id="price"
                            type="number"
                            min={0}
                            step="0.01"
                            value={form.price}
                            onChange={(e) => set("price", e.target.value)}
                            aria-invalid={!!fieldErrors.price}
                            className="h-10"
                          />
                          {fieldErrors.price && <FieldError>{fieldErrors.price}</FieldError>}
                        </Field>
                        <Field>
                          <FieldLabel htmlFor="currency">Currency</FieldLabel>
                          <Select value={form.currency} onValueChange={(v) => set("currency", v)}>
                            <SelectTrigger id="currency" className="h-10">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              {CURRENCIES.map((c) => (
                                <SelectItem key={c} value={c}>
                                  {c}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </Field>
                      </div>
                    )}
                    {isFree && (
                      <p className="text-xs text-muted-foreground">Your event is free. Toggle to add pricing.</p>
                    )}
                  </FieldSet>
                </div>

                <SeparatorLine />

                <div>
                  <h3 className="mb-4 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                    Settings
                  </h3>
                  <FieldSet>
                    <Field data-invalid={!!fieldErrors.capacity}>
                      <FieldLabel htmlFor="sidebar-capacity">Capacity</FieldLabel>
                      <Input
                        id="sidebar-capacity"
                        type="number"
                        min={1}
                        value={form.capacity}
                        onChange={(e) => set("capacity", e.target.value)}
                        aria-invalid={!!fieldErrors.capacity}
                        className="h-10"
                      />
                      {fieldErrors.capacity && <FieldError>{fieldErrors.capacity}</FieldError>}
                    </Field>

                    <Field>
                      <div className="flex items-center justify-between">
                        <FieldLabel htmlFor="requires_approval" className="!w-auto">Require approval</FieldLabel>
                        <Switch
                          id="requires_approval"
                          checked={form.requires_approval}
                          onCheckedChange={(v) => set("requires_approval", v)}
                        />
                      </div>
                    </Field>
                  </FieldSet>
                </div>

                <SeparatorLine />

                <Button
                  type="submit"
                  size="lg"
                  disabled={createMut.isPending}
                  className="w-full"
                >
                  {createMut.isPending && <Spinner data-icon="inline-start" />}
                  Publish event
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => navigate(-1)}
                  className="w-full"
                >
                  Cancel
                </Button>
              </div>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}

function SeparatorLine() {
  return <div className="h-px bg-border" />;
}
