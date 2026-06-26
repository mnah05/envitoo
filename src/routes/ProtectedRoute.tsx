import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "@clerk/clerk-react";
import { Spinner } from "@/components/ui/spinner";

/** Gate that requires a Clerk session. Shows a spinner while auth loads. */
export function ProtectedRoute() {
  const { isLoaded, isSignedIn } = useAuth();
  const location = useLocation();

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
