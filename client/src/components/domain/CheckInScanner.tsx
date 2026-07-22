import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { BrowserMultiFormatReader, type IScannerControls } from "@zxing/browser";
import { Check, QrCode, ScanLine, X } from "lucide-react";
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

  // useLayoutEffect ensures videoRef is populated before we access the camera.
  useLayoutEffect(() => {
    if (!open) {
      stop();
      return;
    }
    setError(null);
    setFlash(null);
    let cancelled = false;

    const video = videoRef.current;

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

    async function start() {
      try {
        // Attempt 1: rear camera on phones
        let stream: MediaStream;
        try {
          stream = await navigator.mediaDevices.getUserMedia({
            video: { facingMode: "environment" },
            audio: false,
          });
        } catch {
          // Fallback: any camera (laptops → front camera)
          stream = await navigator.mediaDevices.getUserMedia({
            video: true,
            audio: false,
          });
        }
        if (cancelled || !video) {
          stream.getTracks().forEach((t) => t.stop());
          return;
        }

        streamRef.current = stream;
        video.srcObject = stream;
        await video.play();

        // Pass the already-playing video to zxing for scanning only
        const reader = new BrowserMultiFormatReader();
        const controls = await reader.decodeFromVideoElement(video, onScanResult);
        if (cancelled) controls.stop();
        else controlsRef.current = controls;
      } catch (err) {
        if (cancelled) return;
        const msg = err instanceof Error ? err.message : String(err);
        console.error("[CheckInScanner]", err);

        if (/permission|notallowed|denied/i.test(msg)) {
          setError("Camera access was denied. Allow camera permissions and try again.");
        } else {
          setError("Could not start the camera. Make sure no other app is using it.");
        }
      }
    }

    start();

    return () => {
      cancelled = true;
      stop();
    };
  }, [open, onResult, stop]);

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
            <p className="text-sm text-muted-foreground">{error}</p>
            <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
              Close
            </Button>
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