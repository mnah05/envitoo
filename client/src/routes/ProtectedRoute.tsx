import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "@clerk/clerk-react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useMe } from "@/hooks/use-me";
import { ApiError } from "@/lib/api";

/** Gate that requires a Clerk session. Shows a spinner while auth loads. */
export function ProtectedRoute() {
  const { isLoaded, isSignedIn } = useAuth();
  const location = useLocation();
  const isWelcome = location.pathname === "/welcome";
  const me = useMe(isLoaded && isSignedIn);

  if (!isLoaded) {
    return (
      <div className="grid min-h-[60vh] place-items-center">
        <Spinner className="size-6 text-muted-foreground" />
      </div>
    );
  }
  if (!isSignedIn) {
    return <Navigate to="/signin" replace state={{ from: location.pathname }} />;
  }

  if (isWelcome) {
    return <Outlet />;
  }

  if (me.isLoading || me.isFetching) {
    return (
      <div className="grid min-h-[60vh] place-items-center">
        <Spinner className="size-6 text-muted-foreground" />
      </div>
    );
  }

  if (me.isError) {
    // Authenticated but not onboarded yet — run the setup flow, which
    // calls POST /api/me to provision the account. A retry button here
    // could never succeed on its own.
    if (me.error instanceof ApiError && me.error.status === 403) {
      return <Navigate to="/welcome" replace />;
    }
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 px-4 text-center">
        <p className="text-sm text-muted-foreground">
          Couldn&apos;t load your account. Please try again.
        </p>
        <Button variant="outline" onClick={() => me.refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  return <Outlet />;
}

/** Redirect signed-in users away from auth pages. */
export function PublicOnlyRoute() {
  const { isLoaded, isSignedIn } = useAuth();
  if (!isLoaded) {
    return (
      <div className="grid min-h-dvh place-items-center">
        <Spinner className="size-6 text-muted-foreground" />
      </div>
    );
  }
  if (isSignedIn) return <Navigate to="/home" replace />;
  return <Outlet />;
}
