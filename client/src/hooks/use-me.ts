import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useApi, getMe, postMe } from "@/lib/api";

export function useMe(enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: ["me"],
    queryFn: () => getMe(api),
    enabled,
    staleTime: 60_000,
  });
}

/**
 * Provisions the local user via POST /api/me (the explicit onboarding
 * entry point) and seeds the ["me"] query cache so downstream useMe
 * callers render without a redundant GET.
 */
export function useProvisionMe() {
  const api = useApi();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => postMe(api),
    onSuccess: (user) => {
      queryClient.setQueryData(["me"], user);
    },
  });
}
