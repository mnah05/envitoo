import { useState, useEffect } from "react";
import { Settings2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Spinner } from "@/components/ui/spinner";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { useUpsertVolunteerSettings } from "@/hooks/use-volunteers";
import type { VolunteerSettings } from "@/lib/api";

export function VolunteerSettingsPanel({
  eventId,
  settings,
}: {
  eventId: string;
  settings: VolunteerSettings | undefined;
}) {
  const upsertMut = useUpsertVolunteerSettings(eventId);

  const [enabled, setEnabled] = useState(false);
  const [description, setDescription] = useState("");
  const [maxApps, setMaxApps] = useState("");

  useEffect(() => {
    if (settings) {
      setEnabled(settings.enabled);
      setDescription(settings.description ?? "");
      setMaxApps(settings.max_applications != null ? String(settings.max_applications) : "");
    }
  }, [settings]);

  function onSave() {
    upsertMut.mutate({
      enabled,
      description: description.trim() || undefined,
      max_applications: maxApps ? Number(maxApps) : undefined,
    });
  }

  return (
    <div className="rounded-2xl border bg-card p-6 shadow-sm">
      <div className="flex items-center gap-2">
        <Settings2 className="size-5 text-muted-foreground" />
        <h3 className="font-semibold">Volunteer settings</h3>
      </div>

      <FieldGroup className="mt-4">
        <div className="flex items-center justify-between rounded-xl border p-4">
          <div>
            <p className="text-sm font-medium">Open volunteer applications</p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              Allow people to apply as volunteers for this event.
            </p>
          </div>
          <Switch checked={enabled} onCheckedChange={setEnabled} />
        </div>

        <Field>
          <FieldLabel htmlFor="vol-desc">Description</FieldLabel>
          <Textarea
            id="vol-desc"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Tell volunteers what you need help with"
            rows={3}
            disabled={!enabled}
            className={!enabled ? "opacity-50" : ""}
          />
          <FieldDescription>Shown on the event page. Up to 2000 characters.</FieldDescription>
        </Field>

        <Field>
          <FieldLabel htmlFor="vol-max">Max applications</FieldLabel>
          <Input
            id="vol-max"
            type="number"
            min={1}
            value={maxApps}
            onChange={(e) => setMaxApps(e.target.value)}
            placeholder="Unlimited"
            disabled={!enabled}
            className={!enabled ? "opacity-50" : ""}
          />
          <FieldDescription>Leave empty for unlimited applications.</FieldDescription>
        </Field>
      </FieldGroup>

      <div className="mt-4">
        <Button onClick={onSave} disabled={upsertMut.isPending}>
          {upsertMut.isPending && <Spinner data-icon="inline-start" />}
          Save settings
        </Button>
      </div>
    </div>
  );
}
