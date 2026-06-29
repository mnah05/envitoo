import { useQuery } from "@tanstack/react-query";
import { useApi, getMe } from "@/lib/api";

export function useMe(enabled = true) {
  const api = useApi();
  return useQuery({
    queryKey: ["me"],
    queryFn: () => getMe(api),
    enabled,
    staleTime: 60_000,
  });
}
