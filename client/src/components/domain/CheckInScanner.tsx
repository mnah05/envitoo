import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { BrowserMultiFormatReader, type IScannerControls } from "@zxing/browser";
import { Check, QrCode, RefreshCw, ScanLine, X } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

/**
 * Extract the access_token (`token` query param) from a scanned QR payload.
 * The QR encodes a ticket URL like `${origin}/ticket?token=<uuid>`.
 * Returns the token, or null if the payload isn't a valid ticket QR.
 */
export function parseTicketToken(raw: string): string | null {
  try {
    const url = new URL(raw);
    if (url.pathname !== "/ticket") return null;
    return url.searchParams.get("token");
  } catch {
    return null;
  }
}

function describeCameraError(err: unknown): string {
  const name = err instanceof DOMException ? err.name : (err as Error)?.name;
  const msg = err instanceof Error ? err.message : String(err);
  const combined = `${name} ${msg}`.toLowerCase();

  // iOS Safari / non-secure contexts: the whole API is missing.
  if (typeof navigator === "undefined" || !navigator.mediaDevices?.getUserMedia) {
    return "Camera is not available here. Make sure you're connected over HTTPS (the padlock in the address bar) and using Safari or Chrome.";
  }
  if (/notallowed|permission|denied/i.test(combined)) {
    return "Camera access was denied. Tap the camera/refresh icon in the address bar to allow it, or enable it in Settings > Privacy > Camera, then try again.";
  }
  if (/notfound/i.test(combined) || /no camera/i.test(combined)) {
    return "No camera was found on this device.";
  }
  if (/notreadable|trackstart|in use/i.test(combined)) {
    return "Couldn't start the camera. Make sure no other app (camera, FaceTime, zoom) is using it, then retry.";
  }
  if (/overconstrained|constraint/i.test(combined)) {
    return "Could not start the camera. Try again — if it keeps failing, close other apps using the camera.";
  }
  return "Could not start the camera. Make sure no other app is using it and that you're on HTTPS, then retry.";
}

export function CheckInScanner({
  open,
  onOpenChange,
  onResult,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onResult: (token: string) => void;
}) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const controlsRef = useRef<IScannerControls | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const cooldownRef = useRef(0);
  const lastTokenRef = useRef<string>("");
  const startRef = useRef<() => void>(() => {});
  const [error, setError] = useState<string | null>(null);
  const [flash, setFlash] = useState<string | null>(null);

  const stop = useCallback(() => {
    controlsRef.current?.stop();
    controlsRef.current = null;
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
  }, []);

  // requestAnimationFrame so getUserMedia runs straight after the tap that
  // opened the dialog — iOS Safari drops camera permission prompts that are
  // not issued within a user gesture.
  useLayoutEffect(() => {
    if (!open) {
      stop();
      setError(null);
      return;
    }
    setError(null);
    setFlash(null);
    requestAnimationFrame(() => startRef.current());
  }, [open, stop]);

  startRef.current = function start() {
    const video = videoRef.current;
    let cancelled = false;
    if (controlsRef.current || streamRef.current) stop();

    const onScanResult = (result: any) => {
      if (cancelled || !result) return;
      const text = result.getText();
      const token = parseTicketToken(text);
      if (!token) return;
      const now = Date.now();
      if (now - cooldownRef.current < 1500) return;
      if (token === lastTokenRef.current && now - cooldownRef.current < 3000) return;
      cooldownRef.current = now;
      lastTokenRef.current = token;
      setFlash(token.slice(0, 8) + "…" + token.slice(-6));
      onResult(token);
    };

    async function run() {
      try {
        let stream: MediaStream;
        try {
          // Attempt 1: rear camera on phones (ideal, not exact — some iOS
          // builds throw OverconstrainedError for an exact facingMode).
          stream = await navigator.mediaDevices.getUserMedia({
            video: { facingMode: { ideal: "environment" } },
            audio: false,
          });
        } catch {
          // Fallback: any camera (laptops → front camera)
          stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
        }

        if (cancelled || !video) {
          stream.getTracks().forEach((t) => t.stop());
          return;
        }

        streamRef.current = stream;
        video.playsInline = true; // required on iOS or the stream is fullscreen
        video.muted = true;
        video.srcObject = stream;
        await video.play();

        // iOS reports a 0x0 frame until the stream is actually rendering;
        // give zxing a ready frame before handing it the element.
        for (let i = 0; i < 50 && video.videoWidth === 0; i++) {
          await new Promise((r) => setTimeout(r, 50));
        }

        const reader = new BrowserMultiFormatReader();
        const controls = await reader.decodeFromVideoElement(video, onScanResult);
        if (cancelled) controls.stop();
        else controlsRef.current = controls;
      } catch (err) {
        if (cancelled) return;
        console.error("[CheckInScanner]", err);
        setError(describeCameraError(err));
        if (streamRef.current) {
          streamRef.current.getTracks().forEach((t) => t.stop());
          streamRef.current = null;
        }
      }
    }

    run();

    // `cancelled` is captured by run() before it resolves; a retry creates a
    // fresh start() so a stale run can flag itself as cancelled via an early
    // closure. Track it with a local so a new start() invalidates the old one.
    startRef.current = function invalidated() {
      cancelled = true;
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((t) => t.stop());
        streamRef.current = null;
      }
    };
  };

  // Brief success overlay after each scan — auto-clears so scanning resumes.
  useEffect(() => {
    if (!flash) return;
    const t = setTimeout(() => setFlash(null), 1500);
    return () => clearTimeout(t);
  }, [flash]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ScanLine className="size-5" /> Scan to check in
          </DialogTitle>
          <DialogDescription>
            Point the camera at an attendee&apos;s ticket QR code. Keep scanning to check
            in multiple people.
          </DialogDescription>
        </DialogHeader>

        {error ? (
          <div className="flex flex-col items-center gap-3 py-8 text-center">
            <div className="grid size-12 place-items-center rounded-full bg-destructive/10 text-destructive">
              <X className="size-6" />
            </div>
            <p className="max-w-sm text-sm text-muted-foreground">{error}</p>
            <div className="mt-1 flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => startRef.current()}>
                <RefreshCw className="size-4" />
                Retry camera
              </Button>
              <Button variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
                Close
              </Button>
            </div>
          </div>
        ) : (
          <div className="relative overflow-hidden rounded-xl border bg-black">
            <video
              ref={videoRef}
              autoPlay
              muted
              playsInline
              className="aspect-square w-full object-cover"
            />
            {flash && (
              <div className="absolute inset-0 grid place-items-center bg-primary/20 backdrop-blur-sm">
                <div className="flex flex-col items-center gap-2 rounded-2xl bg-background/90 px-6 py-4 text-center shadow-lg">
                  <span className="grid size-10 place-items-center rounded-full bg-primary text-primary-foreground">
                    <Check className="size-6" />
                  </span>
                  <p className="font-medium">Checked in</p>
                  <p className="font-mono text-xs text-muted-foreground">{flash}</p>
                </div>
              </div>
            )}
            {!flash && (
              <div className="pointer-events-none absolute inset-0 grid place-items-center">
                <div className="size-48 rounded-xl border-2 border-primary/80 shadow-[0_0_0_2px_rgba(15,15,30,0.8)]" />
              </div>
            )}
          </div>
        )}

        <div className="flex items-center justify-between">
          <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <QrCode className="size-3.5" /> Scan attendee ticket QR codes
          </span>
          <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            Done
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}