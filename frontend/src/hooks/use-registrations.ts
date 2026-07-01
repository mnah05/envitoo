import {
  useMutation,
  useQuery,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { toast } from "sonner";
import {
  useApi,
  createGuestRegistration,
  getGuestRegistration,
  cancelGuestRegistration,
  createRegistration,
  getRegistration,
  cancelRegistration,
  listMyRegistrations,
  listEventRegistrations,
  approveRegistration,
  rejectRegistration,
  updateRegistrationStatus,
  checkInAttendee,
  checkInByEventToken,
  type CreateGuestRegistrationValues,
  type CreateRegistrationValues,
  type CheckInValues,
  type UpdateRegistrationStatusValues,
} from "@/lib/api";
import { ApiError } from "@/lib/api";

export const regKeys = {
  guest: (token: string) => ["registrations", "guest", token] as const,
  detail: (id: string) => ["registrations", "detail", id] as const,
  mine: (opts: { limit?: number; offset?: number }) => ["registrations", "mine", opts] as const,
  event: (eventId: string, opts: { limit?: number; offset?: number }) =>
    ["registrations", "event", eventId, opts] as const,
};

// --- Guest (public) ---------------------------------------------------------

export function useGuestRegistration(token: string, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: regKeys.guest(token),
    queryFn: () => getGuestRegistration(api, token),
    enabled: enabled && !!token,
    retry: (count, err) => (err instanceof ApiError && err.status === 404 ? false : count < 2),
  });
}

export function useCreateGuestRegistration() {
  const api = useApi();
  return useMutation({
    mutationFn: ({ eventId, values }: { eventId: string; values: CreateGuestRegistrationValues }) =>
      createGuestRegistration(api, eventId, values),
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to register");
    },
  });
}

export function useCancelGuestRegistration() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (token: string) => cancelGuestRegistration(api, token),
    onSuccess: (reg) => {
      toast.success("Registration cancelled");
      if (reg.access_token) qc.invalidateQueries({ queryKey: regKeys.guest(reg.access_token) });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to cancel");
    },
  });
}

// --- Attendee (protected) ---------------------------------------------------

export function useRegistration(id: string, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: regKeys.detail(id),
    queryFn: () => getRegistration(api, id),
    enabled: enabled && !!id,
  });
}

export function useMyRegistrations(opts: { limit?: number; offset?: number } = {}, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: regKeys.mine(opts),
    queryFn: () => listMyRegistrations(api, opts),
    enabled,
    placeholderData: keepPreviousData,
  });
}

export function useCreateRegistration() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ eventId, values }: { eventId: string; values: CreateRegistrationValues }) =>
      createRegistration(api, eventId, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["registrations", "mine"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to register");
    },
  });
}

export function useCancelRegistration() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => cancelRegistration(api, id),
    onSuccess: () => {
      toast.success("Registration cancelled");
      qc.invalidateQueries({ queryKey: ["registrations", "mine"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to cancel");
    },
  });
}

// --- Organizer (admin) ------------------------------------------------------

export function useEventRegistrations(
  eventId: string,
  opts: { limit?: number; offset?: number } = {},
  enabled = true,
) {
  const api = useApi();
  return useQuery({
    queryKey: regKeys.event(eventId, opts),
    queryFn: () => listEventRegistrations(api, eventId, opts),
    enabled: enabled && !!eventId,
    placeholderData: keepPreviousData,
  });
}

export function useApproveRegistration() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => approveRegistration(api, id),
    onSuccess: (reg) => {
      toast.success("Attendee approved");
      qc.invalidateQueries({ queryKey: ["registrations", "event", reg.event_id] });
      qc.invalidateQueries({ queryKey: ["events"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to approve");
    },
  });
}

export function useRejectRegistration() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => rejectRegistration(api, id),
    onSuccess: (reg) => {
      toast.success("Attendee rejected");
      qc.invalidateQueries({ queryKey: ["registrations", "event", reg.event_id] });
      qc.invalidateQueries({ queryKey: ["events"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to reject");
    },
  });
}

export function useUpdateRegistrationStatus() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: UpdateRegistrationStatusValues }) =>
      updateRegistrationStatus(api, id, values),
    onSuccess: (reg) => {
      toast.success("Status updated");
      qc.invalidateQueries({ queryKey: ["registrations", "event", reg.event_id] });
      qc.invalidateQueries({ queryKey: ["events"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to update status");
    },
  });
}

export function useCheckIn() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: CheckInValues }) =>
      checkInAttendee(api, id, values),
    onSuccess: (reg) => {
      toast.success("Checked in");
      qc.invalidateQueries({ queryKey: ["registrations", "event", reg.event_id] });
      qc.invalidateQueries({ queryKey: ["events"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Check-in failed");
    },
  });
}

export function useCheckInByEventToken(eventId: string) {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CheckInValues) => checkInByEventToken(api, eventId, values),
    onSuccess: (reg) => {
      toast.success("Checked in");
      qc.invalidateQueries({ queryKey: ["registrations", "event", reg.event_id] });
      qc.invalidateQueries({ queryKey: ["events"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Check-in failed");
    },
  });
}
