import { useCallback, useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import {
  addMonths,
  eachDayOfInterval,
  endOfMonth,
  endOfWeek,
  format,
  isSameDay,
  isSameMonth,
  isToday,
  startOfMonth,
  startOfWeek,
  subMonths,
} from "date-fns";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

interface CalendarProps {
  selected: Date | undefined;
  onSelect: (date: Date) => void;
}

const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export function Calendar({ selected, onSelect }: CalendarProps) {
  const today = new Date();
  const [view, setView] = useState(selected ?? today);

  const days = eachDayOfInterval({
    start: startOfWeek(startOfMonth(view)),
    end: endOfWeek(endOfMonth(view)),
  });

  const prev = useCallback(() => setView((v) => subMonths(v, 1)), []);
  const next = useCallback(() => setView((v) => addMonths(v, 1)), []);

  return (
    <div className="w-full">
      {/* Header */}
      <div className="mb-3 flex items-center justify-between">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          onClick={prev}
        >
          <ChevronLeft className="size-4" />
        </Button>
        <span className="text-sm font-semibold">
          {format(view, "MMMM yyyy")}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          onClick={next}
        >
          <ChevronRight className="size-4" />
        </Button>
      </div>

      {/* Day names */}
      <div className="mb-1 grid grid-cols-7">
        {DAYS.map((d) => (
          <div
            key={d}
            className="py-1 text-center text-xs font-medium text-muted-foreground"
          >
            {d}
          </div>
        ))}
      </div>

      {/* Day grid */}
      <div className="grid grid-cols-7">
        {days.map((day) => {
          const sameMonth = isSameMonth(day, view);
          const picked = selected && isSameDay(day, selected);
          const todayDay = isToday(day);

          return (
            <button
              key={day.toISOString()}
              type="button"
              onClick={() => onSelect(day)}
              className={cn(
                "flex h-9 w-full items-center justify-center rounded-md text-sm transition-colors",
                !sameMonth && "text-muted-foreground/40",
                sameMonth && !picked && "hover:bg-muted hover:text-foreground",
                picked && "bg-primary text-primary-foreground hover:bg-primary/90",
                todayDay && !picked && "border border-primary/50",
              )}
            >
              {format(day, "d")}
            </button>
          );
        })}
      </div>
    </div>
  );
}
