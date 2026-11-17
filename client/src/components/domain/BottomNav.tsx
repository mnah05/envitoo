import { NavLink } from "react-router-dom";
import { useAuth } from "@clerk/clerk-react";
import { CalendarDays, Compass, Plus, Ticket, UserRound } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * Mobile-only bottom navigation bar.
 *
 * Signed-out: Discover · Create · Sign in
 * Signed-in:  Discover · Create · My tickets · My events
 *
 * Hidden on sm+ (the desktop header handles nav there).
 */
export function BottomNav() {
  const { isLoaded, isSignedIn } = useAuth();
  return (
    <nav
      className={cn(
        "fixed inset-x-0 bottom-0 z-40 border-t border-border bg-background/95 backdrop-blur-md sm:hidden",
        "pb-[env(safe-area-inset-bottom)]",
      )}
      aria-label="Primary"
    >
      <div className="mx-auto flex h-14 max-w-md items-stretch justify-around">
        <NavItem to="/discover" label="Discover" icon={<Compass className="size-5" />} />

        <NavLink
          to="/create"
          className="flex flex-col items-center justify-center px-3"
          aria-label="Create event"
        >
          {({ isActive }) => (
            <span
              className={cn(
                "grid size-10 -translate-y-1.5 place-items-center rounded-full bg-primary text-primary-foreground transition-colors active:bg-primary/90",
                isActive && "ring-2 ring-primary/25 ring-offset-2 ring-offset-background",
              )}
            >
              <Plus className="size-5" />
            </span>
          )}
        </NavLink>

        {isLoaded && isSignedIn ? (
          <>
          <NavItem to="/home" label="Tickets" icon={<Ticket className="size-5" />} />
          <NavItem to="/manage" label="Events" icon={<CalendarDays className="size-5" />} />
          </>
        ) : (
          <NavLink
            to="/signin"
            className="flex flex-1 flex-col items-center justify-center gap-1 text-muted-foreground transition-colors hover:text-foreground"
            aria-label="Sign in"
          >
            <UserRound className="size-5" />
            <span className="text-[11px] font-medium">Sign in</span>
          </NavLink>
        )}
      </div>
    </nav>
  );
}

function NavItem({
  to,
  label,
  icon,
}: {
  to: string;
  label: string;
  icon: React.ReactNode;
}) {
  return (
    <NavLink
      to={to}
      className="flex flex-1 flex-col items-center justify-center gap-1 text-muted-foreground transition-colors"
    >
      {({ isActive }) => (
        <>
          <span className={cn("transition-colors", isActive && "text-foreground")}>
            {icon}
          </span>
          <span
            className={cn(
              "text-[11px] font-medium",
              isActive ? "text-foreground" : "text-muted-foreground",
            )}
          >
            {label}
          </span>
        </>
      )}
    </NavLink>
  );
}
