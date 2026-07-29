import { Dumbbell, Globe, Palette } from "lucide-react";
import {
  CpuIcon,
  GraduationCapIcon,
  RocketIcon,
  WrenchIcon,
} from "lucide-animated";
import { cn } from "@/lib/utils";

/** Category chip icon — animated where available, static Lucide otherwise. */
export function CategoryIcon({
  category,
  className,
  size = 16,
  animateOnHover = true,
}: {
  category: string;
  className?: string;
  size?: number;
  animateOnHover?: boolean;
}) {
  const animatedClass = cn("text-primary", className);
  const staticClass = cn("size-4 text-primary", className);

  switch (category) {
    case "technical":
      return <CpuIcon size={size} className={animatedClass} animateOnHover={animateOnHover} />;
    case "workshop":
      return <WrenchIcon size={size} className={animatedClass} animateOnHover={animateOnHover} />;
    case "seminar":
      return (
        <GraduationCapIcon size={size} className={animatedClass} animateOnHover={animateOnHover} />
      );
    case "hackathon":
      return <RocketIcon size={size} className={animatedClass} animateOnHover={animateOnHover} />;
    case "cultural":
      return <Palette className={staticClass} strokeWidth={1.5} />;
    case "sports":
      return <Dumbbell className={staticClass} strokeWidth={1.5} />;
    case "other":
    default:
      return <Globe className={staticClass} strokeWidth={1.5} />;
  }
}
