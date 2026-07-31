import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useApi, listMyRegistrations } from "@/lib/api";
import { useProvisionMe } from "@/hooks/use-me";
import { regKeys } from "@/hooks/use-registrations";

export const SETUP_STEPS = [
  { id: "verifying", label: "Verifying your account" },
  { id: "profile", label: "Setting up your profile" },
  { id: "linking", label: "Linking your tickets" },
  { id: "ready", label: "You're all set" },
] as const;

const MIN_DURATION_MS = 2500;
const STEP_INTERVAL_MS = 700;

export function useAccountSetup() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const api = useApi();
  const { mutate, isSuccess, isError, error } = useProvisionMe();

  const [activeStepIndex, setActiveStepIndex] = useState(0);
  const startTimeRef = useRef(Date.now());
  const navigatedRef = useRef(false);
  const startedRef = useRef(false);

  // Fire the provisioning request once on mount (the ref guards against
  // StrictMode double-mount). POST /api/me is idempotent, so retrying
  // after an error is safe.
  useEffect(() => {
    if (startedRef.current) return;
    startedRef.current = true;
    mutate();
  }, [mutate]);

  useEffect(() => {
    if (isSuccess) {
      setActiveStepIndex(SETUP_STEPS.length - 1);
      return;
    }

    const interval = setInterval(() => {
      setActiveStepIndex((prev) => {
        const maxWhileLoading = SETUP_STEPS.length - 2;
        return prev >= maxWhileLoading ? prev : prev + 1;
      });
    }, STEP_INTERVAL_MS);

    return () => clearInterval(interval);
  }, [isSuccess]);

  useEffect(() => {
    if (!isSuccess || navigatedRef.current) return;

    const elapsed = Date.now() - startTimeRef.current;
    const remaining = Math.max(0, MIN_DURATION_MS - elapsed);

    const timer = setTimeout(async () => {
      if (navigatedRef.current) return;
      navigatedRef.current = true;

      try {
        const regs = await queryClient.fetchQuery({
          queryKey: regKeys.mine({ limit: 50 }),
          queryFn: () => listMyRegistrations(api, { limit: 50 }),
        });
        const count = regs.registrations?.length ?? 0;
        if (count > 0) {
          toast.success(
            count === 1
              ? "Your ticket is ready in your dashboard"
              : `${count} tickets are ready in your dashboard`,
          );
        }
      } catch {
        // Non-blocking — dashboard will fetch registrations on its own.
      }

      navigate("/home", { replace: true });
    }, remaining);

    return () => clearTimeout(timer);
  }, [isSuccess, navigate, queryClient, api]);

  return {
    steps: SETUP_STEPS,
    activeStepIndex,
    // Pending covers idle (pre-mount) and in-flight states so the
    // spinner shows from the first frame.
    isPending: !isSuccess && !isError,
    isError,
    error,
    refetch: mutate,
  };
}