import { useState } from "react";
import { CalendarIcon } from "lucide-react";
import { Popover as PopoverPrimitive } from "radix-ui";
import { format } from "date-fns";
import { cn } from "@/lib/utils";
import { Calendar } from "@/components/ui/calendar";

interface DateTimePickerProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  id?: string;
  "aria-invalid"?: boolean;
}

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = ["00", "15", "30", "45"];

export function DateTimePicker({
  value,
  onChange,
  placeholder = "Pick a date & time",
  className,
  id,
  "aria-invalid": ariaInvalid,
}: DateTimePickerProps) {
  const [open, setOpen] = useState(false);
  const date = value ? new Date(value) : undefined;

  function handleDateSelect(day: Date) {
    const d = date ?? new Date();
    day.setHours(d.getHours(), d.getMinutes());
    onChange(day.toISOString().slice(0, 16));
  }

  function handleTimeChange(hours: number, minutes: string) {
    const d = date ?? new Date();
    d.setHours(hours, Number(minutes), 0, 0);
    onChange(d.toISOString().slice(0, 16));
  }

  const selectedHours = date?.getHours() ?? 12;
  const selectedMinutes = date ? String(date.getMinutes()).padStart(2, "0") : "00";

  return (
    <PopoverPrimitive.Root open={open} onOpenChange={setOpen}>
      <PopoverPrimitive.Trigger asChild>
        <button
          type="button"
          id={id}
          aria-invalid={ariaInvalid}
          className={cn(
            "flex h-10 w-full items-center justify-between gap-2 rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs transition-[color,box-shadow] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:bg-input/30 dark:aria-invalid:ring-destructive/40",
            !value && "text-muted-foreground",
            className,
          )}
        >
          <span className="flex items-center gap-2">
            <CalendarIcon className="size-4 shrink-0 text-muted-foreground" />
            {value ? format(new Date(value), "MMM d, yyyy  h:mm a") : placeholder}
          </span>
        </button>
      </PopoverPrimitive.Trigger>

      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          align="start"
          sideOffset={6}
          className={cn(
            "z-50 w-[280px] rounded-2xl border border-border bg-card p-4 shadow-lg outline-none",
            "data-[state=open]:animate-in data-[state=closed]:animate-out",
            "data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
            "data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
            "data-[side=bottom]:slide-in-from-top-2",
          )}
        >
          <Calendar selected={date} onSelect={handleDateSelect} />

          {/* Time picker */}
          <div className="mt-4 flex items-center gap-2 border-t border-border pt-4">
            <span className="mr-1 text-xs font-medium text-muted-foreground">Time</span>
            <select
              value={selectedHours}
              onChange={(e) => handleTimeChange(Number(e.target.value), selectedMinutes)}
              className="h-8 flex-1 rounded-md border border-input bg-transparent px-2 text-sm outline-none focus:border-ring focus:ring-[3px] focus:ring-ring/50"
            >
              {HOURS.map((h) => (
                <option key={h} value={h}>
                  {String(h).padStart(2, "0")}:00
                </option>
              ))}
            </select>
            <select
              value={selectedMinutes}
              onChange={(e) => handleTimeChange(selectedHours, e.target.value)}
              className="h-8 flex-1 rounded-md border border-input bg-transparent px-2 text-sm outline-none focus:border-ring focus:ring-[3px] focus:ring-ring/50"
            >
              {MINUTES.map((m) => (
                <option key={m} value={m}>
                  :{m}
                </option>
              ))}
            </select>
          </div>
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  );
}
