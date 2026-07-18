import { useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { SignedIn, SignedOut, SignInButton, UserButton } from "@clerk/clerk-react";
import { CalendarDays, Compass, Menu, Plus, Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ThemeToggle } from "@/components/domain/ThemeToggle";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

function BrandMark({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "grid size-7 place-items-center rounded-lg bg-primary text-primary-foreground text-xs font-bold tracking-tight",
        className,
      )}
    >
      E
    </span>
  );
}

export function Header() {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <header className="fixed inset-x-0 top-0 z-50 border-b border-border/50 bg-background/80 pt-[env(safe-area-inset-top)] backdrop-blur-lg">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between gap-4 px-4 sm:h-16 sm:px-6">
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="-ml-2 sm:hidden"
            aria-label="Open menu"
            onClick={() => setMobileOpen(true)}
          >
            <Menu className="size-5" />
          </Button>

          <Link
            to="/"
            className="flex items-center gap-2.5 text-foreground"
          >
            <BrandMark />
            <span className="font-display text-[15px] font-semibold tracking-[-0.02em] sm:text-[16px]">
              EnviToo
            </span>
          </Link>
        </div>

        <nav className="hidden items-center gap-0.5 sm:flex">
          <HeaderLink to="/discover">Discover</HeaderLink>
          <SignedIn>
            <HeaderLink to="/home">My tickets</HeaderLink>
            <HeaderLink to="/manage">My events</HeaderLink>
          </SignedIn>
        </nav>

        <div className="flex items-center gap-1">
          <ThemeToggle />
          <SignedIn>
            <Button asChild size="sm" className="hidden sm:inline-flex">
              <Link to="/create">
                <Plus className="size-4" />
                Create event
              </Link>
            </Button>
            <UserButton afterSignOutUrl="/" />
          </SignedIn>

          <SignedOut>
            <SignInButton mode="redirect" forceRedirectUrl="/home">
              <Button variant="ghost" size="sm" className="text-muted-foreground hover:text-foreground">
                Sign in
              </Button>
            </SignInButton>
            <Button asChild size="sm" className="hidden sm:inline-flex">
              <Link to="/create">
                <Plus className="size-4" />
                Create event
              </Link>
            </Button>
          </SignedOut>
        </div>
      </div>

      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="bottom" className="sm:hidden">
          <SheetHeader>
            <SheetTitle className="text-left">Browse EnviToo</SheetTitle>
          </SheetHeader>
          <nav className="flex flex-col gap-1 px-2 pb-4">
            <MobileLink
              to="/discover"
              label="Discover"
              icon={<Compass className="size-5" />}
              onClick={() => setMobileOpen(false)}
            />
            <SignedIn>
              <MobileLink
                to="/home"
                label="My tickets"
                icon={<Ticket className="size-5" />}
                onClick={() => setMobileOpen(false)}
              />
              <MobileLink
                to="/manage"
                label="My events"
                icon={<CalendarDays className="size-5" />}
                onClick={() => setMobileOpen(false)}
              />
              <MobileLink
                to="/create"
                label="Create event"
                icon={<Plus className="size-5" />}
                onClick={() => setMobileOpen(false)}
              />
            </SignedIn>
            <SignedOut>
              <MobileLink
                to="/create"
                label="Create event"
                icon={<Plus className="size-5" />}
                onClick={() => setMobileOpen(false)}
              />
              <div className="mt-2 px-3">
                <SignInButton mode="redirect" forceRedirectUrl="/home">
                  <Button className="w-full">Sign in</Button>
                </SignInButton>
              </div>
            </SignedOut>
          </nav>
        </SheetContent>
      </Sheet>
    </header>
  );
}

function HeaderLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        cn(
          "text-ui rounded-md px-3 py-1.5 text-[13px] text-muted-foreground transition-colors hover:bg-primary/10 hover:text-foreground",
          isActive && "bg-primary/15 text-primary",
        )
      }
    >
      {children}
    </NavLink>
  );
}

function MobileLink({
  to,
  label,
  icon,
  onClick,
}: {
  to: string;
  label: string;
  icon: React.ReactNode;
  onClick: () => void;
}) {
  return (
    <NavLink
      to={to}
      onClick={onClick}
      className={({ isActive }) =>
        cn(
          "flex min-h-[48px] items-center gap-3 rounded-xl px-3 text-base font-medium transition-colors",
          isActive
            ? "bg-primary/15 text-primary"
            : "text-foreground hover:bg-muted/60",
        )
      }
    >
      {icon}
      {label}
    </NavLink>
  );
}
