import { ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export function RoleBadge({
  isAdmin,
  className,
}: {
  isAdmin: boolean;
  className?: string;
}) {
  if (!isAdmin) return null;
  return (
    <Badge
      variant="outline"
      className={cn(
        "bg-primary/15 text-primary ring-1 ring-primary/30",
        className,
      )}
    >
      <ShieldCheck className="size-3" />
      Admin
    </Badge>
  );
}
