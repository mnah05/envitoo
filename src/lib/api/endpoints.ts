import {
  EventListSchema,
  EventSchema,
  RegistrationListSchema,
  RegistrationSchema,
  UserSchema,
  type CreateEventValues,
  type CreateGuestRegistrationValues,
  type CreateRegistrationValues,
  type UpdateEventValues,
  type UpdateRegistrationStatusValues,
  type CheckInValues,
  type Event,
  type EventList,
  type Registration,
  type RegistrationList,
  type User,
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
