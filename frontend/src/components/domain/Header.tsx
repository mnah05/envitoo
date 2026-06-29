import { useEffect, useState } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";
import { SignedIn, SignedOut, SignInButton, UserButton, useUser } from "@clerk/clerk-react";
import { Activity, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function Header() {
  const [scrolled, setScrolled] = useState(false);
  const [hidden, setHidden] = useState(false);
  const { pathname } = useLocation();
  const { user } = useUser();
  const isLanding = pathname === "/";

  useEffect(() => {
    let last = window.scrollY;
    const onScroll = () => {
      const y = window.scrollY;
      setScrolled(y > 8);
      setHidden(y > 240 && y > last);
      last = y;
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const solid = scrolled || !isLanding;

  return (
    <header
      className={cn(
        "fixed inset-x-0 top-0 z-50 transition-transform duration-300",
        hidden && "-translate-y-full",
      )}
    >
      <div
        className={cn(
          "transition-colors duration-300",
          solid
            ? "border-b bg-background/85 backdrop-blur-md"
            : "border-b border-transparent bg-transparent",
        )}
      >
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
          <Link
            to="/"
            className="flex items-center gap-2 font-bold tracking-tight text-foreground"
          >
            <span className="grid size-7 place-items-center rounded-lg bg-primary text-primary-foreground">
              <Activity className="size-4" strokeWidth={2.5} />
            </span>
            <span className="text-lg">Pulse</span>
          </Link>

          <nav className="hidden items-center gap-1 sm:flex">
            <HeaderLink to="/discover">Discover events</HeaderLink>
            <SignedIn>
              <HeaderLink to="/home">My tickets</HeaderLink>
              <HeaderLink to="/manage">My events</HeaderLink>
            </SignedIn>
          </nav>

          <div className="flex items-center gap-2">
            <SignedIn>
              <Button asChild size="sm" className="hidden sm:inline-flex">
                <Link to="/create">
                  <Plus data-icon="inline-start" />
                  Create event
                </Link>
              </Button>
              <div className="flex items-center gap-2">
                <div className="hidden text-right sm:block">
                  <p className="text-xs text-muted-foreground">Signed in</p>
                  <p className="max-w-[10rem] truncate text-sm font-medium">
                    {user?.username || user?.firstName || user?.emailAddresses?.[0]?.emailAddress}
                  </p>
                </div>
                <UserButton afterSignOutUrl="/" />
              </div>
            </SignedIn>

            <SignedOut>
              <SignInButton mode="redirect" forceRedirectUrl="/home">
                <Button variant="ghost" size="sm">
                  Sign in
                </Button>
              </SignInButton>
              <Button asChild size="sm">
                <Link to="/create">
                  <Plus data-icon="inline-start" />
                  Create event
                </Link>
              </Button>
            </SignedOut>
          </div>
        </div>
      </div>
    </header>
  );
}

function HeaderLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        cn(
          "rounded-full px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground",
          isActive && "text-foreground",
        )
      }
    >
      {children}
    </NavLink>
  );
}
