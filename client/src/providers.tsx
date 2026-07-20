import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ClerkProvider } from "@clerk/clerk-react";
import { ThemeProvider } from "next-themes";
import { Toaster } from "@/components/ui/sonner";
import { useClerkAppearance } from "@/lib/clerk-appearance";

const clerkPubKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY;

if (!clerkPubKey) {
  // eslint-disable-next-line no-console
  console.warn("Missing VITE_CLERK_PUBLISHABLE_KEY — sign-in will not work.");
}

function ClerkWithTheme({ children }: { children: React.ReactNode }) {
  const appearance = useClerkAppearance();
  return (
    <ClerkProvider
      publishableKey={clerkPubKey!}
      signInUrl="/signin"
      signUpUrl="/signup"
      appearance={appearance}
    >
      {children}
    </ClerkProvider>
  );
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: 1,
            refetchOnWindowFocus: false,
            staleTime: 30_000,
          },
        },
      }),
  );

  return (
    <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false}>
      <ClerkWithTheme>
        <QueryClientProvider client={client}>
          {children}
          <Toaster position="top-center" richColors closeButton />
        </QueryClientProvider>
      </ClerkWithTheme>
    </ThemeProvider>
  );
}
