import { z } from "zod";

/*
  Zod schemas mirroring the Go backend's JSON shapes.
  - Events & registrations use camelCase DTO tags (dto/event.go, dto/registration.go).
  - /api/me returns the raw db.User with PascalCase fields (no json tags), so the
    User schema uses PascalCase to match what the server actually emits.
*/

const uuid = z.string().uuid();
const iso = z.string().datetime({ offset: true }).or(z.string().datetime());

export const eventStatus = z.enum(["published", "cancelled", "completed"]);
export const regStatus = z.enum(["pending", "approved", "rejected", "cancelled"]);
export const category = z.enum([
  "technical",
  "cultural",
  "sports",
  "workshop",
  "seminar",
  "hackathon",
  "other",
]);

export const EventSchema = z.object({
  id: uuid,
  organizer_id: uuid,
  title: z.string(),
  description: z.string().nullable().optional(),
  starts_at: iso,
  ends_at: iso,
  capacity: z.number().int(),
  price_in_paise: z.number().int(),
  currency: z.string().length(3),
  status: eventStatus,
  slug: z.string(),
  category: category.nullable().optional(),
  requires_approval: z.boolean(),
  cover_image_url: z.string().url().nullable().optional(),
  calendar_id: uuid.nullable().optional(),
  is_virtual: z.boolean(),
  meeting_url: z.string().url().nullable().optional(),
  venue_name: z.string().nullable().optional(),
  lat: z.number().nullable().optional(),
  lng: z.number().nullable().optional(),
  created_at: iso,
  updated_at: iso,
});
export type Event = z.infer<typeof EventSchema>;

export const EventListSchema = z.object({
  events: z.array(EventSchema),
  limit: z.number().int(),
  offset: z.number().int(),
  total: z.number().int(),
});
export type EventList = z.infer<typeof EventListSchema>;

export const RegistrationSchema = z.object({
  id: uuid,
  event_id: uuid,
  user_id: uuid.nullable().optional(),
  name: z.string(),
  email: z.string(),
  status: regStatus,
  access_token: uuid.optional(),
  token_expires_at: iso.nullable().optional(),
  checked_in_at: iso.nullable().optional(),
  created_at: iso,
  updated_at: iso,
});
export type Registration = z.infer<typeof RegistrationSchema>;

export const RegistrationListSchema = z.object({
  registrations: z.array(RegistrationSchema),
  limit: z.number().int(),
  offset: z.number().int(),
  total: z.number().int(),
});
export type RegistrationList = z.infer<typeof RegistrationListSchema>;

/** /api/me returns the raw db.User (PascalCase, no json tags). */
export const UserSchema = z.object({
  ID: uuid,
  AuthUserID: z.string(),
  Email: z.string(),
  Username: z.string().nullable().optional(),
  AvatarUrl: z.string().nullable().optional(),
  CreatedAt: iso,
  UpdatedAt: iso,
});
export type User = z.infer<typeof UserSchema>;

export const CurrencySchema = z.string().length(3).default("INR");

export const CreateEventInput = z
  .object({
    title: z.string().min(1).max(200),
    description: z.string().max(5000).optional(),
    starts_at: iso,
    ends_at: iso,
    capacity: z.number().int().min(1),
    price_in_paise: z.number().int().min(0).default(0),
    currency: CurrencySchema,
    requires_approval: z.boolean().default(false),
    cover_image_url: z.string().url().optional(),
    calendar_id: uuid.optional(),
    is_virtual: z.boolean(),
    meeting_url: z.string().url().optional(),
    venue_name: z.string().max(200).optional(),
    lat: z.number().optional(),
    lng: z.number().optional(),
    category: category.optional(),
  })
  .refine(
    (d) => (d.is_virtual ? !!d.meeting_url : !!d.venue_name && d.lat != null && d.lng != null),
    { message: "virtual events need meeting_url; physical events need venue_name, lat, lng" },
  );
export type CreateEventValues = z.infer<typeof CreateEventInput>;

export const UpdateEventInput = z.object({
  title: z.string().min(1).max(200).optional(),
  description: z.string().max(5000).optional(),
  starts_at: iso.optional(),
  ends_at: iso.optional(),
  capacity: z.number().int().min(1).optional(),
  price_in_paise: z.number().int().min(0).optional(),
  currency: z.string().length(3).optional(),
  requires_approval: z.boolean().optional(),
  cover_image_url: z.string().url().optional(),
  is_virtual: z.boolean().optional(),
  meeting_url: z.string().url().optional(),
  venue_name: z.string().max(200).optional(),
  lat: z.number().optional(),
  lng: z.number().optional(),
  category: category.optional(),
});
export type UpdateEventValues = z.infer<typeof UpdateEventInput>;

export const CreateGuestRegistrationInput = z.object({
  name: z.string().min(1).max(200),
  email: z.string().email(),
});
export type CreateGuestRegistrationValues = z.infer<typeof CreateGuestRegistrationInput>;

export const CreateRegistrationInput = z.object({
  name: z.string().min(1).max(200).optional(),
  email: z.string().email().optional(),
});
export type CreateRegistrationValues = z.infer<typeof CreateRegistrationInput>;

export const UpdateRegistrationStatusInput = z.object({
  status: regStatus,
});
export type UpdateRegistrationStatusValues = z.infer<typeof UpdateRegistrationStatusInput>;

export const CheckInInput = z.object({
  access_token: uuid,
});
export type CheckInValues = z.infer<typeof CheckInInput>;

export const CATEGORY_LABELS: Record<string, string> = {
  technical: "Tech",
  cultural: "Arts & Culture",
  sports: "Fitness",
  workshop: "Workshop",
  seminar: "Seminar",
  hackathon: "Hackathon",
  other: "Other",
};

export const CATEGORIES = Object.keys(CATEGORY_LABELS) as (keyof typeof CATEGORY_LABELS)[];
