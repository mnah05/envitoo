import {
  useMutation,
  useQuery,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { toast } from "sonner";
import {
  useApi,
  getVolunteerSettings,
  listVolunteerRoles,
  applyVolunteer,
  listMyVolunteerApplications,
  listMyEventStaff,
  cancelVolunteerApplication,
  upsertVolunteerSettings,
  createVolunteerRole,
  deleteVolunteerRole,
  listEventVolunteerApplications,
  listEventVolunteers,
  approveVolunteerApplication,
  rejectVolunteerApplication,
  type UpsertVolunteerSettingsValues,
  type CreateVolunteerRoleValues,
  type CreateVolunteerApplicationValues,
} from "@/lib/api";
import { ApiError } from "@/lib/api";

export const volunteerKeys = {
  settings: (eventId: string) => ["volunteer", "settings", eventId] as const,
  roles: (eventId: string) => ["volunteer", "roles", eventId] as const,
  myApps: (opts: { limit?: number; offset?: number }) =>
    ["volunteer-applications", "mine", opts] as const,
  myStaff: (opts: { limit?: number; offset?: number }) =>
    ["event-staff", "mine", opts] as const,
  eventApps: (eventId: string, opts: { limit?: number; offset?: number }) =>
    ["volunteer", "applications", eventId, opts] as const,
  eventVolunteers: (eventId: string, opts: { limit?: number; offset?: number }) =>
    ["volunteer", "roster", eventId, opts] as const,
};

// --- Queries ----------------------------------------------------------------

export function useVolunteerSettings(eventId: string, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: volunteerKeys.settings(eventId),
    queryFn: () => getVolunteerSettings(api, eventId),
    enabled: enabled && !!eventId,
    retry: (failureCount, error) => {
      if (error instanceof ApiError && error.status === 404) return false;
      return failureCount < 2;
    },
  });
}

export function useVolunteerRoles(eventId: string, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: volunteerKeys.roles(eventId),
    queryFn: () => listVolunteerRoles(api, eventId),
    enabled: enabled && !!eventId,
  });
}

export function useMyVolunteerApplications(opts: { limit?: number; offset?: number } = {}) {
  const api = useApi();
  return useQuery({
    queryKey: volunteerKeys.myApps(opts),
    queryFn: () => listMyVolunteerApplications(api, opts),
  });
}

export function useMyEventStaff(opts: { limit?: number; offset?: number } = {}, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: volunteerKeys.myStaff(opts),
    queryFn: () => listMyEventStaff(api, opts),
    enabled,
  });
}

export function useEventVolunteerApplications(
  eventId: string,
  opts: { limit?: number; offset?: number } = {},
  enabled = true,
) {
  const api = useApi();
  return useQuery({
    queryKey: volunteerKeys.eventApps(eventId, opts),
    queryFn: () => listEventVolunteerApplications(api, eventId, opts),
    enabled: enabled && !!eventId,
    placeholderData: keepPreviousData,
  });
}

export function useEventVolunteers(
  eventId: string,
  opts: { limit?: number; offset?: number } = {},
  enabled = true,
) {
  const api = useApi();
  return useQuery({
    queryKey: volunteerKeys.eventVolunteers(eventId, opts),
    queryFn: () => listEventVolunteers(api, eventId, opts),
    enabled: enabled && !!eventId,
  });
}

// --- Mutations --------------------------------------------------------------

export function useApplyVolunteer(eventId: string) {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateVolunteerApplicationValues) =>
      applyVolunteer(api, eventId, values),
    onSuccess: () => {
      toast.success("Application submitted");
      qc.invalidateQueries({ queryKey: ["volunteer-applications", "mine"] });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) return;
      toast.error(err instanceof ApiError ? err.message : "Failed to apply");
    },
  });
}

export function useCancelVolunteerApplication() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (appId: string) => cancelVolunteerApplication(api, appId),
    onSuccess: () => {
      toast.success("Application cancelled");
      qc.invalidateQueries({ queryKey: ["volunteer-applications", "mine"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to cancel application");
    },
  });
}

export function useUpsertVolunteerSettings(eventId: string) {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: UpsertVolunteerSettingsValues) =>
      upsertVolunteerSettings(api, eventId, values),
    onSuccess: () => {
      toast.success("Volunteer settings saved");
      qc.invalidateQueries({ queryKey: volunteerKeys.settings(eventId) });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to save settings");
    },
  });
}

export function useCreateVolunteerRole(eventId: string) {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateVolunteerRoleValues) => createVolunteerRole(api, eventId, values),
    onSuccess: () => {
      toast.success("Role created");
      qc.invalidateQueries({ queryKey: volunteerKeys.roles(eventId) });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) return;
      toast.error(err instanceof ApiError ? err.message : "Failed to create role");
    },
  });
}

export function useDeleteVolunteerRole(eventId: string) {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (roleId: string) => deleteVolunteerRole(api, eventId, roleId),
    onSuccess: () => {
      toast.success("Role deleted");
      qc.invalidateQueries({ queryKey: volunteerKeys.roles(eventId) });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to delete role");
    },
  });
}

export function useApproveVolunteerApplication(eventId: string) {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (appId: string) => approveVolunteerApplication(api, appId),
    onSuccess: () => {
      toast.success("Application approved");
      qc.invalidateQueries({ queryKey: ["volunteer", "applications", eventId] });
      qc.invalidateQueries({ queryKey: ["volunteer", "roster", eventId] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to approve application");
    },
  });
}

export function useRejectVolunteerApplication(eventId: string) {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (appId: string) => rejectVolunteerApplication(api, appId),
    onSuccess: () => {
      toast.success("Application rejected");
      qc.invalidateQueries({ queryKey: ["volunteer", "applications", eventId] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to reject application");
    },
  });
}
