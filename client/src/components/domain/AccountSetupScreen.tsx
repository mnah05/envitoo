import { useEffect, useState } from "react";
import { useClerk } from "@clerk/clerk-react";
import { AnimatePresence, motion } from "framer-motion";
import { Check, Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";
import { useAccountSetup } from "@/hooks/use-account-setup";

function usePrefersReducedMotion() {
  const [reduced, setReduced] = useState(
    () => typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches,
  );

  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const onChange = () => setReduced(mq.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);

  return reduced;
}

export function AccountSetupScreen() {
  const { signOut } = useClerk();
  const reducedMotion = usePrefersReducedMotion();
  const { steps, activeStepIndex, isPending, isError, refetch } = useAccountSetup();

  if (isError) {
    return (
      <div className="relative flex min-h-dvh flex-col items-center justify-center bg-muted/40 px-4 py-16">
        <div className="pointer-events-none absolute inset-0 bg-light-teal opacity-70" aria-hidden />
        <div className="relative w-full max-w-md rounded-2xl border border-border bg-card p-8 text-center">
          <h1 className="display-title text-xl">Setup interrupted</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            We couldn&apos;t finish setting up your account. Please try again.
          </p>
          <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-center">
            <Button onClick={() => refetch()}>Try again</Button>
            <Button variant="outline" onClick={() => signOut({ redirectUrl: "/signin" })}>
              Sign out
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="relative flex min-h-dvh flex-col items-center justify-center bg-muted/40 px-4 py-16">
      <div className="pointer-events-none absolute inset-0 bg-light-teal" aria-hidden />

      <motion.div
        initial={reducedMotion ? false : { opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: reducedMotion ? 0 : 0.4, ease: "easeOut" }}
        className="relative w-full max-w-md rounded-2xl border border-border bg-card p-8"
      >
        <div className="flex flex-col items-center text-center">
          <div className="relative flex size-16 items-center justify-center">
            {!reducedMotion && (
              <motion.div
                className="absolute inset-0 rounded-full border border-border"
                animate={{ scale: [1, 1.08, 1], opacity: [0.5, 0.9, 0.5] }}
                transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
              />
            )}
            <div className="flex size-14 items-center justify-center rounded-full border border-border/80 bg-muted">
              {isPending ? (
                <Spinner className="size-7 text-foreground" />
              ) : (
                <Check className="size-7 text-foreground" aria-hidden />
              )}
            </div>
          </div>

          <h1 className="display-title mt-6 text-2xl">Setting up your account</h1>
          <p className="text-lede mt-2 text-[1.0625rem]">
            Hang tight — we&apos;re getting everything ready for you.
          </p>
        </div>

        <ol className="mt-8 space-y-3" aria-label="Setup progress">
          {steps.map((step, index) => {
            const isComplete = index < activeStepIndex;
            const isActive = index === activeStepIndex;

            return (
              <motion.li
                key={step.id}
                initial={reducedMotion ? false : { opacity: 0, x: -8 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{
                  duration: reducedMotion ? 0 : 0.3,
                  delay: reducedMotion ? 0 : index * 0.08,
                }}
                className={cn(
                  "flex items-center gap-3 rounded-xl border px-4 py-3 text-sm transition-colors",
                  isActive && "border-primary/40 bg-primary/5 text-foreground",
                  isComplete && "border-border/60 text-muted-foreground",
                  !isActive && !isComplete && "border-transparent text-muted-foreground/60",
                )}
              >
                <span
                  className={cn(
                    "flex size-6 shrink-0 items-center justify-center rounded-full border text-xs",
                    isComplete && "border-primary/50 bg-primary/15 text-primary",
                    isActive && "border-primary bg-primary/10 text-primary",
                    !isActive && !isComplete && "border-border bg-muted/50",
                  )}
                  aria-hidden
                >
                  {isComplete ? (
                    <Check className="size-3.5" />
                  ) : isActive && isPending ? (
                    <Spinner className="size-3.5" />
                  ) : isActive ? (
                    <Check className="size-3.5" />
                  ) : (
                    <Ticket className="size-3 opacity-40" />
                  )}
                </span>
                <span className={cn(isActive && "font-medium")}>{step.label}</span>
              </motion.li>
            );
          })}
        </ol>

        <AnimatePresence>
          {activeStepIndex >= steps.length - 1 && !isPending && (
            <motion.p
              initial={reducedMotion ? false : { opacity: 0 }}
              animate={{ opacity: 1 }}
              className="mt-6 text-center text-sm text-primary"
            >
              Taking you to your dashboard…
            </motion.p>
          )}
        </AnimatePresence>
      </motion.div>
    </div>
  );
}