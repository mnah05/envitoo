import { z } from "zod";
import {
  EventListSchema,
  EventSchema,
  RegistrationListSchema,
  RegistrationSchema,
  UserSchema,
  VolunteerSettingsSchema,
  VolunteerRoleSchema,
  VolunteerApplicationSchema,
  VolunteerApplicationListSchema,
  VolunteerListSchema,
  EventStaffListSchema,
  type CreateEventValues,
  type CreateGuestRegistrationValues,
  type CreateRegistrationValues,
  type UpdateEventValues,
  type UpdateRegistrationStatusValues,
  type CheckInValues,
  type UpsertVolunteerSettingsValues,
  type CreateVolunteerRoleValues,
  type CreateVolunteerApplicationValues,
  type Event,
  type EventList,
  type Registration,
  type RegistrationList,
  type User,
  type VolunteerSettings,
  type VolunteerRole,
  type VolunteerApplication,
  type VolunteerApplicationList,
  type VolunteerList,
  type EventStaffList,
} from "./schemas";
import type { Api } from "./client";

async function unwrap<T>(p: Promise<T>): Promise<T> {
  return p;
}
void unwrap;

function parseOrThrow<T>(schema: { parse: (x: unknown) => T }, data: unknown): T {
  return schema.parse(data);
}

// --- Users ------------------------------------------------------------------

export function getMe(api: Api): Promise<User> {
  return api.get<unknown>("/api/me").then((d) => parseOrThrow(UserSchema, d));
}

// --- Events (public) --------------------------------------------------------

export interface PublicEventsParams {
  category?: string;
  isVirtual?: boolean;
  q?: string;
  startsAfter?: string;
  startsBefore?: string;
  limit?: number;
  offset?: number;
}

export function listPublicEvents(api: Api, params: PublicEventsParams = {}): Promise<EventList> {
  return api
    .get<unknown>("/api/events", {
      category: params.category,
      is_virtual: params.isVirtual === undefined ? undefined : String(params.isVirtual),
      q: params.q,
      starts_after: params.startsAfter,
      starts_before: params.startsBefore,
      limit: params.limit,
      offset: params.offset,
    })
    .then((d) => parseOrThrow(EventListSchema, d));
}

export function getEvent(api: Api, id: string): Promise<Event> {
  return api.get<unknown>(`/api/events/${id}`).then((d) => parseOrThrow(EventSchema, d));
}

export function getEventBySlug(api: Api, slug: string): Promise<Event> {
  return api
    .get<unknown>(`/api/events/slug/${slug}`)
    .then((d) => parseOrThrow(EventSchema, d));
}

// --- Events (organizer) -----------------------------------------------------

export function createEvent(api: Api, values: CreateEventValues): Promise<Event> {
  return api.post<unknown>("/api/events", values).then((d) => parseOrThrow(EventSchema, d));
}

export function listMyEvents(
  api: Api,
  opts: { limit?: number; offset?: number; includeCancelled?: boolean } = {},
): Promise<EventList> {
  return api
    .get<unknown>("/api/me/events", {
      limit: opts.limit,
      offset: opts.offset,
      include_cancelled: opts.includeCancelled ? "true" : undefined,
    })
    .then((d) => parseOrThrow(EventListSchema, d));
}

export function updateEvent(api: Api, id: string, values: UpdateEventValues): Promise<Event> {
  return api
    .patch<unknown>(`/api/events/${id}`, values)
    .then((d) => parseOrThrow(EventSchema, d));
}

export function cancelEvent(api: Api, id: string): Promise<Event> {
  return api.del<unknown>(`/api/events/${id}`).then((d) => parseOrThrow(EventSchema, d));
}

// --- Registrations (guest / public) ----------------------------------------

export function createGuestRegistration(
  api: Api,
  eventId: string,
  values: CreateGuestRegistrationValues,
): Promise<Registration> {
  return api
    .post<unknown>(`/api/events/${eventId}/register/guest`, values)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function getGuestRegistration(api: Api, token: string): Promise<Registration> {
  return api
    .get<unknown>("/api/guest-registrations", { token })
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function cancelGuestRegistration(api: Api, token: string): Promise<Registration> {
  return api
    .del<unknown>("/api/guest-registrations", { token })
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

// --- Registrations (attendee) -----------------------------------------------

export function createRegistration(
  api: Api,
  eventId: string,
  values: CreateRegistrationValues,
): Promise<Registration> {
  return api
    .post<unknown>(`/api/events/${eventId}/register`, values)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function getRegistration(api: Api, id: string): Promise<Registration> {
  return api
    .get<unknown>(`/api/registrations/${id}`)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function cancelRegistration(api: Api, id: string): Promise<Registration> {
  return api
    .del<unknown>(`/api/registrations/${id}`)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function listMyRegistrations(
  api: Api,
  opts: { limit?: number; offset?: number } = {},
): Promise<RegistrationList> {
  return api
    .get<unknown>("/api/me/registrations", { limit: opts.limit, offset: opts.offset })
    .then((d) => parseOrThrow(RegistrationListSchema, d));
}

// --- Registrations (organizer) ---------------------------------------------

export function listEventRegistrations(
  api: Api,
  eventId: string,
  opts: { limit?: number; offset?: number } = {},
): Promise<RegistrationList> {
  return api
    .get<unknown>(`/api/events/${eventId}/registrations`, {
      limit: opts.limit,
      offset: opts.offset,
    })
    .then((d) => parseOrThrow(RegistrationListSchema, d));
}

export function getRegistrationAsAdmin(api: Api, id: string): Promise<Registration> {
  return api
    .get<unknown>(`/api/registrations/${id}/admin`)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function approveRegistration(api: Api, id: string): Promise<Registration> {
  return api
    .post<unknown>(`/api/registrations/${id}/approve`)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function rejectRegistration(api: Api, id: string): Promise<Registration> {
  return api
    .post<unknown>(`/api/registrations/${id}/reject`)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function updateRegistrationStatus(
  api: Api,
  id: string,
  values: UpdateRegistrationStatusValues,
): Promise<Registration> {
  return api
    .patch<unknown>(`/api/registrations/${id}/status`, values)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

export function checkInAttendee(api: Api, id: string, values: CheckInValues): Promise<Registration> {
  return api
    .post<unknown>(`/api/registrations/${id}/check-in`, values)
    .then((d) => parseOrThrow(RegistrationSchema, d));
}

// --- Volunteer — public -----------------------------------------------------

export function getVolunteerSettings(api: Api, eventId: string): Promise<VolunteerSettings> {
  return api
    .get<unknown>(`/api/events/${eventId}/volunteer/settings`)
    .then((d) => parseOrThrow(VolunteerSettingsSchema, d));
}

export function listVolunteerRoles(api: Api, eventId: string): Promise<VolunteerRole[]> {
  return api
    .get<unknown>(`/api/events/${eventId}/volunteer/roles`)
    .then((d) => parseOrThrow(z.array(VolunteerRoleSchema), d));
}

// --- Volunteer — user applications ------------------------------------------

export function applyVolunteer(
  api: Api,
  eventId: string,
  values: CreateVolunteerApplicationValues,
): Promise<VolunteerApplication> {
  return api
    .post<unknown>(`/api/events/${eventId}/volunteer/apply`, values)
    .then((d) => parseOrThrow(VolunteerApplicationSchema, d));
}

export function listMyVolunteerApplications(
  api: Api,
  opts: { limit?: number; offset?: number } = {},
): Promise<VolunteerApplicationList> {
  return api
    .get<unknown>("/api/me/volunteer-applications", { limit: opts.limit, offset: opts.offset })
    .then((d) => parseOrThrow(VolunteerApplicationListSchema, d));
}

export function listMyEventStaff(
  api: Api,
  opts: { limit?: number; offset?: number } = {},
): Promise<EventStaffList> {
  return api
    .get<unknown>("/api/me/event-staff", { limit: opts.limit, offset: opts.offset })
    .then((d) => parseOrThrow(EventStaffListSchema, d));
}

export function getMyVolunteerApplication(api: Api, appId: string): Promise<VolunteerApplication> {
  return api
    .get<unknown>(`/api/volunteer-applications/${appId}`)
    .then((d) => parseOrThrow(VolunteerApplicationSchema, d));
}

export function cancelVolunteerApplication(api: Api, appId: string): Promise<VolunteerApplication> {
  return api
    .del<unknown>(`/api/volunteer-applications/${appId}`)
    .then((d) => parseOrThrow(VolunteerApplicationSchema, d));
}

// --- Volunteer — organizer management ---------------------------------------

export function upsertVolunteerSettings(
  api: Api,
  eventId: string,
  values: UpsertVolunteerSettingsValues,
): Promise<VolunteerSettings> {
  return api
    .put<unknown>(`/api/events/${eventId}/volunteer/settings`, values)
    .then((d) => parseOrThrow(VolunteerSettingsSchema, d));
}

export function createVolunteerRole(
  api: Api,
  eventId: string,
  values: CreateVolunteerRoleValues,
): Promise<VolunteerRole> {
  return api
    .post<unknown>(`/api/events/${eventId}/volunteer/roles`, values)
    .then((d) => parseOrThrow(VolunteerRoleSchema, d));
}

export function deleteVolunteerRole(
  api: Api,
  eventId: string,
  roleId: string,
): Promise<VolunteerRole> {
  return api
    .del<unknown>(`/api/events/${eventId}/volunteer/roles/${roleId}`)
    .then((d) => parseOrThrow(VolunteerRoleSchema, d));
}

// --- Volunteer — staff management -------------------------------------------

export function listEventVolunteerApplications(
  api: Api,
  eventId: string,
  opts: { limit?: number; offset?: number } = {},
): Promise<VolunteerApplicationList> {
  return api
    .get<unknown>(`/api/events/${eventId}/volunteer/applications`, {
      limit: opts.limit,
      offset: opts.offset,
    })
    .then((d) => parseOrThrow(VolunteerApplicationListSchema, d));
}

export function listEventVolunteers(
  api: Api,
  eventId: string,
  opts: { limit?: number; offset?: number } = {},
): Promise<VolunteerList> {
  return api
    .get<unknown>(`/api/events/${eventId}/volunteers`, {
      limit: opts.limit,
      offset: opts.offset,
    })
    .then((d) => parseOrThrow(VolunteerListSchema, d));
}

export function getVolunteerApplicationAsAdmin(
  api: Api,
  appId: string,
): Promise<VolunteerApplication> {
  return api
    .get<unknown>(`/api/volunteer-applications/${appId}/admin`)
    .then((d) => parseOrThrow(VolunteerApplicationSchema, d));
}

export function approveVolunteerApplication(api: Api, appId: string): Promise<VolunteerApplication> {
  return api
    .post<unknown>(`/api/volunteer-applications/${appId}/approve`)
    .then((d) => parseOrThrow(VolunteerApplicationSchema, d));
}

export function rejectVolunteerApplication(api: Api, appId: string): Promise<VolunteerApplication> {
  return api
    .post<unknown>(`/api/volunteer-applications/${appId}/reject`)
    .then((d) => parseOrThrow(VolunteerApplicationSchema, d));
}
