import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useEffect } from "react";
import { LocateFixed } from "lucide-react";
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
  FieldLegend,
} from "@/components/ui/field";
import { PageContainer } from "@/components/domain/Layout";
import { useCreateEvent } from "@/hooks/use-events";
import { CreateEventInput, CATEGORIES, CATEGORY_LABELS, type CreateEventValues } from "@/lib/api";

const CURRENCIES = ["INR", "USD", "EUR", "GBP", "AED", "SGD", "AUD", "CAD", "JPY"];

interface FormState {
  title: string;
  description: string;
  starts_at: string; // datetime-local
  ends_at: string;
  capacity: string;
  price: string; // major units
  currency: string;
  requires_approval: boolean;
  cover_image_url: string;
  is_virtual: string; // "physical" | "virtual"
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

  useEffect(() => {
    document.title = "Create event — Pulse";
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
    <div className="pt-16">
      <PageContainer width="narrow" className="py-10">
        <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">Create an event</h1>
        <p className="mt-2 text-muted-foreground">
          Set up your event page. You can edit details later.
        </p>

        <form onSubmit={onSubmit} className="mt-8">
          <FieldGroup>
            <Field data-invalid={!!fieldErrors.title}>
              <FieldLabel htmlFor="title">Event title</FieldLabel>
              <Input
                id="title"
                value={form.title}
                onChange={(e) => set("title", e.target.value)}
                placeholder="e.g. AI Builders Meetup"
                aria-invalid={!!fieldErrors.title}
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
                rows={5}
              />
              <FieldDescription>Markdown-free plain text. Up to 5000 characters.</FieldDescription>
            </Field>

            <FieldSet>
              <FieldLegend>Date &amp; capacity</FieldLegend>
              <div className="grid gap-5 sm:grid-cols-2">
                <Field data-invalid={!!fieldErrors.starts_at}>
                  <FieldLabel htmlFor="starts_at">Starts</FieldLabel>
                  <Input
                    id="starts_at"
                    type="datetime-local"
                    value={form.starts_at}
                    onChange={(e) => set("starts_at", e.target.value)}
                    aria-invalid={!!fieldErrors.starts_at}
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
                  />
                  {fieldErrors.ends_at && <FieldError>{fieldErrors.ends_at}</FieldError>}
                </Field>
                <Field data-invalid={!!fieldErrors.capacity}>
                  <FieldLabel htmlFor="capacity">Capacity</FieldLabel>
                  <Input
                    id="capacity"
                    type="number"
                    min={1}
                    value={form.capacity}
                    onChange={(e) => set("capacity", e.target.value)}
                    aria-invalid={!!fieldErrors.capacity}
                  />
                  {fieldErrors.capacity && <FieldError>{fieldErrors.capacity}</FieldError>}
                </Field>
                <Field data-invalid={!!fieldErrors.category}>
                  <FieldLabel htmlFor="category">Category</FieldLabel>
                  <Select value={form.category} onValueChange={(v) => set("category", v)}>
                    <SelectTrigger id="category">
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
              </div>
            </FieldSet>

            <FieldSet>
              <FieldLegend>Location</FieldLegend>
              <Field>
                <FieldLabel>Event format</FieldLabel>
                <ToggleGroup
                  type="single"
                  value={form.is_virtual}
                  onValueChange={(v) => v && set("is_virtual", v)}
                  variant="outline"
                  className="w-full"
                >
                  <ToggleGroupItem value="physical" className="flex-1">
                    In person
                  </ToggleGroupItem>
                  <ToggleGroupItem value="virtual" className="flex-1">
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

            <FieldSet>
              <FieldLegend>Tickets</FieldLegend>
              <div className="grid gap-5 sm:grid-cols-3">
                <Field data-invalid={!!fieldErrors.price}>
                  <FieldLabel htmlFor="price">Price ({form.currency})</FieldLabel>
                  <Input
                    id="price"
                    type="number"
                    min={0}
                    step="0.01"
                    value={form.price}
                    onChange={(e) => set("price", e.target.value)}
                    aria-invalid={!!fieldErrors.price}
                  />
                  {fieldErrors.price ? (
                    <FieldError>{fieldErrors.price}</FieldError>
                  ) : (
                    <FieldDescription>0 for free events.</FieldDescription>
                  )}
                </Field>
                <Field>
                  <FieldLabel htmlFor="currency">Currency</FieldLabel>
                  <Select value={form.currency} onValueChange={(v) => set("currency", v)}>
                    <SelectTrigger id="currency">
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
                <Field>
                  <FieldLabel htmlFor="requires_approval">Approval required</FieldLabel>
                  <div className="flex h-9 items-center gap-3">
                    <Switch
                      id="requires_approval"
                      checked={form.requires_approval}
                      onCheckedChange={(v) => set("requires_approval", v)}
                    />
                    <span className="text-sm text-muted-foreground">
                      {form.requires_approval ? "Review each signup" : "Auto-approve"}
                    </span>
                  </div>
                </Field>
              </div>
            </FieldSet>

            <Field data-invalid={!!fieldErrors.cover_image_url}>
              <FieldLabel htmlFor="cover_image_url">Cover image URL</FieldLabel>
              <Input
                id="cover_image_url"
                type="url"
                value={form.cover_image_url}
                onChange={(e) => set("cover_image_url", e.target.value)}
                placeholder="https://…/cover.jpg"
                aria-invalid={!!fieldErrors.cover_image_url}
              />
              {fieldErrors.cover_image_url ? (
                <FieldError>{fieldErrors.cover_image_url}</FieldError>
              ) : (
                <FieldDescription>Optional. A 16:9 image works best.</FieldDescription>
              )}
            </Field>
          </FieldGroup>

          <div className="mt-8 flex items-center gap-3">
            <Button type="submit" size="lg" disabled={createMut.isPending}>
              {createMut.isPending && <Spinner data-icon="inline-start" />}
              Publish event
            </Button>
            <Button type="button" variant="ghost" size="lg" onClick={() => navigate(-1)}>
              Cancel
            </Button>
          </div>
        </form>
      </PageContainer>
    </div>
  );
}
