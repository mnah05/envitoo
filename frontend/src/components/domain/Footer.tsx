import { Link } from "react-router-dom";
import { SignedIn, SignedOut, SignInButton } from "@clerk/clerk-react";

function BrandMark() {
  return (
    <span className="grid size-6 place-items-center rounded-md bg-gradient-to-br from-primary via-teal-400 to-sky-400 text-primary-foreground text-[11px] font-bold tracking-tight shadow-sm">
      E
    </span>
  );
}

export function Footer() {
  return (
    <footer className="relative border-t border-border/60 bg-background">
      <div
        className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/40 to-transparent"
        aria-hidden
      />
      <div
        className="pointer-events-none absolute inset-0 bg-grain opacity-[0.025]"
        aria-hidden
      />

      <div className="relative mx-auto w-full max-w-6xl px-4 py-12 pb-28 sm:px-6 sm:pb-12">
        <div className="flex flex-col gap-10 sm:flex-row sm:items-start sm:justify-between">
          <div className="max-w-xs">
            <Link to="/" className="inline-flex items-center gap-2">
              <BrandMark />
              <span className="font-display text-[15px] font-semibold tracking-[-0.02em]">
                EnviToo
              </span>
            </Link>
            <p className="mt-3 text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
              Host and discover events people actually show up for.
            </p>
          </div>

          <nav className="flex flex-wrap gap-x-10 gap-y-6 text-[0.9375rem] tracking-[-0.01em] text-muted-foreground">
            <div className="flex flex-col gap-2">
              <Link to="/discover" className="transition-colors hover:text-foreground">
                Discover
              </Link>
              <Link to="/create" className="transition-colors hover:text-foreground">
                Create event
              </Link>
            </div>
            <div className="flex flex-col gap-2">
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
                    className="text-left transition-colors hover:text-foreground"
                  >
                    Sign in
                  </button>
                </SignInButton>
                <Link to="/signup" className="transition-colors hover:text-foreground">
                  Sign up
                </Link>
              </SignedOut>
            </div>
          </nav>
        </div>

        <div className="mt-12 flex flex-col gap-2 border-t border-border/60 pt-6 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-eyebrow">© {new Date().getFullYear()} EnviToo</p>
          <p className="text-eyebrow text-[12px]">
            Events · tickets · volunteers · check-in
          </p>
        </div>
      </div>
    </footer>
  );
}
