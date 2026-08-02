import {
  useMutation,
  useQuery,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { toast } from "sonner";
import {
  useApi,
  listPublicEvents,
  getEvent,
  getEventBySlug,
  createEvent,
  listMyEvents,
  updateEvent,
  cancelEvent,
  type PublicEventsParams,
  type CreateEventValues,
  type UpdateEventValues,
  type Event,
} from "@/lib/api";
import { ApiError } from "@/lib/api";

export const eventsKeys = {
  public: (params: PublicEventsParams) => ["events", "public", params] as const,
  detail: (id: string) => ["events", "detail", id] as const,
  slug: (slug: string) => ["events", "slug", slug] as const,
  mine: (opts: { includeCancelled?: boolean }) => ["events", "mine", opts] as const,
};

export function usePublicEvents(params: PublicEventsParams = {}) {
  const api = useApi();
  return useQuery({
    queryKey: eventsKeys.public(params),
    queryFn: () => listPublicEvents(api, params),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
}

export function useEvent(id: string, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: eventsKeys.detail(id),
    queryFn: () => getEvent(api, id),
    enabled: enabled && !!id,
  });
}

export function useEventBySlug(slug: string, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: eventsKeys.slug(slug),
    queryFn: () => getEventBySlug(api, slug),
    enabled: enabled && !!slug,
    retry: (failureCount, error) => {
      if (error instanceof ApiError && error.status === 404) return false;
      return failureCount < 2;
    },
  });
}

export function useMyEvents(opts: { includeCancelled?: boolean } = {}, enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: eventsKeys.mine(opts),
    queryFn: () => listMyEvents(api, opts),
    enabled,
  });
}

export function useCreateEvent() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateEventValues) => createEvent(api, values),
    onSuccess: (event: Event) => {
      if (event.warning) {
        toast.warning(`Event created, but: ${event.warning}`);
      } else {
        toast.success(`"${event.title}" published`);
      }
      qc.invalidateQueries({ queryKey: ["events", "mine"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to create event");
    },
  });
}

export function useUpdateEvent() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: UpdateEventValues }) =>
      updateEvent(api, id, values),
    onSuccess: (event: Event) => {
      if (event.warning) {
        toast.warning(`Event updated, but: ${event.warning}`);
      } else {
        toast.success("Event updated");
      }
      qc.invalidateQueries({ queryKey: ["events"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to update event");
    },
  });
}

export function useCancelEvent() {
  const api = useApi();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => cancelEvent(api, id),
    onSuccess: (event: Event) => {
      toast.success(`"${event.title}" cancelled`);
      qc.invalidateQueries({ queryKey: ["events"] });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Failed to cancel event");
    },
  });
}
