import { Link } from "react-router-dom";
import { SignedIn, SignedOut, SignInButton } from "@clerk/clerk-react";

function BrandMark() {
  return (
    <span className="grid size-6 place-items-center rounded-md bg-gradient-to-br from-primary via-teal-400 to-sky-400 text-primary-foreground text-[11px] font-bold tracking-tight shadow-sm shadow-primary/30">
      E
    </span>
  );
}

export function Footer() {
  return (
    <footer className="border-t border-border/60 bg-background">
      <div className="mx-auto w-full max-w-6xl px-4 py-12 pb-28 sm:px-6 sm:pb-12">
        <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
          <div className="sm:col-span-2 lg:col-span-1">
            <Link to="/" className="inline-flex items-center gap-2">
              <BrandMark />
              <span className="font-display text-[15px] font-semibold tracking-[-0.02em]">
                EnviToo
              </span>
            </Link>
            <p className="mt-3 max-w-xs text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
              Host and discover events people actually show up for.
            </p>
          </div>

          <div>
            <p className="text-ui text-[13px] text-foreground">Product</p>
            <nav className="mt-3 flex flex-col gap-2 text-[0.9375rem] tracking-[-0.01em] text-muted-foreground">
              <Link to="/discover" className="transition-colors hover:text-foreground">
                Discover
              </Link>
              <Link to="/create" className="transition-colors hover:text-foreground">
                Create event
              </Link>
            </nav>
          </div>

          <div>
            <p className="text-ui text-[13px] text-foreground">Account</p>
            <nav className="mt-3 flex flex-col gap-2 text-[0.9375rem] tracking-[-0.01em] text-muted-foreground">
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
            </nav>
          </div>

          <div>
            <p className="text-ui text-[13px] text-foreground">Get started</p>
            <p className="mt-3 text-[0.9375rem] leading-[1.5] tracking-[-0.01em] text-muted-foreground">
              Publish an event page in minutes. Free to start — no subscriptions.
            </p>
            <Link
              to="/create"
              className="text-ui mt-3 inline-flex text-[13px] text-foreground underline-offset-4 hover:underline"
            >
              Create your first event →
            </Link>
          </div>
        </div>

        <div className="mt-12 flex flex-col gap-2 border-t border-border/60 pt-6 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-eyebrow">
            © {new Date().getFullYear()} EnviToo
          </p>
          <p className="text-eyebrow text-[12px]">
            Events · tickets · volunteers · check-in
          </p>
        </div>
      </div>
    </footer>
  );
}
