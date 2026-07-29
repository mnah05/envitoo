import { Link } from "react-router-dom";
import { SignedIn, SignedOut, SignInButton } from "@clerk/clerk-react";

export function Footer() {
  return (
    <footer className="relative border-t border-border bg-background">
      <div className="mx-auto w-full max-w-6xl px-4 pb-6 pt-12 sm:px-6 sm:pb-4 sm:pt-14">
        <div className="flex flex-col gap-10 sm:flex-row sm:items-end sm:justify-between">
          <div className="max-w-sm">
            <Link to="/" className="inline-flex items-center gap-2 text-foreground">
              <span className="grid size-6 place-items-center rounded-md bg-primary text-[11px] font-semibold tracking-tight text-primary-foreground">
                E
              </span>
              <span className="font-display text-base font-semibold tracking-[-0.03em]">
                EnviToo
              </span>
            </Link>
            <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
              Host and discover events people actually show up for.
            </p>
          </div>

          <div className="flex flex-wrap gap-x-8 gap-y-3 text-sm text-muted-foreground">
            <Link to="/discover" className="transition-colors hover:text-foreground">
              Discover
            </Link>
            <Link to="/create" className="transition-colors hover:text-foreground">
              Create event
            </Link>
            <SignedIn>
              <Link to="/home" className="transition-colors hover:text-foreground">
                My tickets
              </Link>
              <Link to="/manage" className="transition-colors hover:text-foreground">
                My events
              </Link>
            </SignedIn>
            <SignedOut>
              <SignInButton mode="redirect" forceRedirectUrl="/home">
                <button
                  type="button"
                  className="cursor-pointer transition-colors hover:text-foreground"
                >
                  Sign in
                </button>
              </SignInButton>
              <Link to="/signup" className="transition-colors hover:text-foreground">
                Sign up
              </Link>
            </SignedOut>
          </div>
        </div>

        <div className="mt-10 border-t border-border pt-5 sm:flex sm:items-center sm:justify-between">
          <p className="text-xs text-muted-foreground">
            &copy; {new Date().getFullYear()} EnviToo
          </p>
        </div>
      </div>
    </footer>
  );
}
