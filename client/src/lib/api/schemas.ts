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
export const eventVisibility = z.enum(["public", "unlisted"]);
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
  registration_count: z.number().int(),
  price_in_paise: z.number().int(),
  currency: z.string().length(3),
  status: eventStatus,
  visibility: eventVisibility,
  slug: z.string(),
  category: category.nullable().optional(),
  requires_approval: z.boolean(),
  cover_image_url: z.string().url().nullable().optional(),
  calendar_id: uuid.nullable().optional(),
  is_virtual: z.boolean(),
  meeting_url: z.string().url().nullable().optional(),
  venue_name: z.string().nullable().optional(),
  location_address: z.string().nullable().optional(),
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

/** /api/me returns the UserResponse DTO (snake_case json tags). */
export const UserSchema = z.object({
  id: uuid,
  auth_user_id: z.string(),
  email: z.string(),
  username: z.string().nullable().optional(),
  avatar_url: z.string().nullable().optional(),
  created_at: iso,
  updated_at: iso,
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
    location_address: z.string().max(500).optional(),
    lat: z.number().optional(),
    lng: z.number().optional(),
    category: category.optional(),
    visibility: eventVisibility.optional(),
  })
  .refine(
    (d) =>
      d.is_virtual
        ? !!d.meeting_url
        : !!d.venue_name && !!d.location_address && d.lat != null && d.lng != null,
    {
      message:
        "virtual events need meeting_url; physical events need venue_name, location_address, lat, lng",
    },
  );
export type CreateEventValues = z.infer<typeof CreateEventInput>;

export const UpdateEventInput = z.object({
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
  location_address: z.string().max(500).optional(),
  lat: z.number().optional(),
  lng: z.number().optional(),
  category: category.optional(),
  visibility: eventVisibility.optional(),
});
export type UpdateEventValues = z.infer<typeof UpdateEventInput>;

export const CreateGuestRegistrationInput = z.object({
  name: z.string().min(1).max(200),
  email: z.string().email(),
});
export type CreateGuestRegistrationValues = z.infer<typeof CreateGuestRegistrationInput>;

export const RequestGuestOTPInput = z.object({
  name: z.string().min(1).max(200),
  email: z.string().email(),
});
export type RequestGuestOTPValues = z.infer<typeof RequestGuestOTPInput>;

export const RequestGuestOTPResponseSchema = z.object({
  status: z.string(),
  expires_in_seconds: z.number().int(),
});
export type RequestGuestOTPResponse = z.infer<typeof RequestGuestOTPResponseSchema>;

export const VerifyGuestOTPInput = z.object({
  email: z.string().email(),
  otp: z.string().regex(/^\d{6}$/, "Enter the 6-digit code"),
});
export type VerifyGuestOTPValues = z.infer<typeof VerifyGuestOTPInput>;

export const VerifyGuestOTPResponseSchema = z.object({
  status: z.string(),
  expires_in_seconds: z.number().int(),
});
export type VerifyGuestOTPResponse = z.infer<typeof VerifyGuestOTPResponseSchema>;

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

// --- Volunteer schemas ------------------------------------------------------

export const volunteerAppStatus = z.enum(["pending", "approved", "rejected", "cancelled"]);

export const VolunteerSettingsSchema = z.object({
  event_id: uuid,
  enabled: z.boolean(),
  description: z.string().nullable().optional(),
  max_applications: z.number().int().nullable().optional(),
  created_at: iso,
  updated_at: iso,
});
export type VolunteerSettings = z.infer<typeof VolunteerSettingsSchema>;

export const VolunteerRoleSchema = z.object({
  id: uuid,
  event_id: uuid,
  name: z.string(),
  is_admin: z.boolean(),
  created_at: iso,
});
export type VolunteerRole = z.infer<typeof VolunteerRoleSchema>;

export const VolunteerApplicationSchema = z.object({
  id: uuid,
  event_id: uuid,
  user_id: uuid,
  role_id: uuid,
  role_name: z.string(),
  is_admin: z.boolean(),
  name: z.string(),
  email: z.string(),
  phone: z.string().nullable().optional(),
  description: z.string(),
  status: volunteerAppStatus,
  created_at: iso,
  updated_at: iso,
});
export type VolunteerApplication = z.infer<typeof VolunteerApplicationSchema>;

export const VolunteerApplicationListSchema = z.object({
  applications: z.array(VolunteerApplicationSchema),
  limit: z.number().int(),
  offset: z.number().int(),
  total: z.number().int(),
});
export type VolunteerApplicationList = z.infer<typeof VolunteerApplicationListSchema>;

export const VolunteerSchema = z.object({
  id: uuid,
  event_id: uuid,
  user_id: uuid,
  role: z.string(),
  email: z.string(),
  username: z.string().nullable().optional(),
  avatar_url: z.string().nullable().optional(),
  created_at: iso,
});
export type Volunteer = z.infer<typeof VolunteerSchema>;

export const VolunteerListSchema = z.object({
  volunteers: z.array(VolunteerSchema),
  limit: z.number().int(),
  offset: z.number().int(),
  total: z.number().int(),
});
export type VolunteerList = z.infer<typeof VolunteerListSchema>;

export const EventStaffItemSchema = z.object({
  event_id: uuid,
  title: z.string(),
  slug: z.string(),
  starts_at: iso,
  ends_at: iso,
  event_status: eventStatus,
  application_id: uuid,
  role_id: uuid,
  role_name: z.string(),
  is_admin: z.boolean(),
  approved_at: iso,
});
export type EventStaffItem = z.infer<typeof EventStaffItemSchema>;

export const EventStaffListSchema = z.object({
  events: z.array(EventStaffItemSchema),
  limit: z.number().int(),
  offset: z.number().int(),
  total: z.number().int(),
});
export type EventStaffList = z.infer<typeof EventStaffListSchema>;

// --- Volunteer request schemas ----------------------------------------------

export const UpsertVolunteerSettingsInput = z.object({
  enabled: z.boolean(),
  description: z.string().max(2000).optional(),
  max_applications: z.number().int().min(1).nullable().optional(),
});
export type UpsertVolunteerSettingsValues = z.infer<typeof UpsertVolunteerSettingsInput>;

export const CreateVolunteerRoleInput = z.object({
  name: z.string().min(1).max(100),
  is_admin: z.boolean(),
});
export type CreateVolunteerRoleValues = z.infer<typeof CreateVolunteerRoleInput>;

export const CreateVolunteerApplicationInput = z.object({
  role_id: uuid,
  name: z.string().min(1).max(200).optional(),
  email: z.string().email().optional(),
  phone: z.string().max(20).optional(),
  description: z.string().min(1).max(5000),
});
export type CreateVolunteerApplicationValues = z.infer<typeof CreateVolunteerApplicationInput>;

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
