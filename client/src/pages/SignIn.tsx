import { SignIn } from "@clerk/clerk-react";
import { useLocation } from "react-router-dom";

export function SignInPage() {
  const location = useLocation();
  const redirect =
    (location.state as { from?: string } | null)?.from ?? "/home";

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-muted/40 px-4 py-16">
      <SignIn
        routing="path"
        path="/signin"
        signUpUrl="/signup"
        fallbackRedirectUrl={redirect}
      />
    </div>
  );
}
