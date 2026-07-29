import { SignUp } from "@clerk/clerk-react";
import { useLocation } from "react-router-dom";

export function SignUpPage() {
  const location = useLocation();
  const redirect =
    (location.state as { from?: string } | null)?.from ?? "/welcome";

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-background px-4 py-16">
      <SignUp
        routing="path"
        path="/signup"
        signInUrl="/signin"
        fallbackRedirectUrl={redirect}
      />
    </div>
  );
}
