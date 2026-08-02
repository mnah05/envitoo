import { useCallback, useEffect, useRef, useState } from "react";
import { useDropzone } from "react-dropzone";
import { ImageIcon, Upload, X } from "lucide-react";
import { cn } from "@/lib/utils";

type Props = {
  onChange: (file: File | null) => void;
  error?: string;
  previewUrl?: string;
};

const MAX_SIZE = 5 * 1024 * 1024; // 5 MB

export function ImageUpload({ onChange, error, previewUrl }: Props) {
  const [preview, setPreview] = useState<string | null>(previewUrl ?? null);
  const previewRef = useRef<string | null>(previewUrl ?? null);

  useEffect(() => {
    return () => revoke(previewRef.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const revoke = (url: string | null) => {
    if (url) URL.revokeObjectURL(url);
  };

  const setPreviewURL = (url: string | null) => {
    revoke(previewRef.current);
    previewRef.current = url;
    setPreview(url);
  };

  const onDrop = useCallback(
    (accepted: File[]) => {
      const file = accepted[0];
      if (!file) return;
      onChange(file);
      setPreviewURL(URL.createObjectURL(file));
    },
    [onChange],
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: { "image/jpeg": [".jpg", ".jpeg"], "image/png": [".png"], "image/webp": [".webp"] },
    maxSize: MAX_SIZE,
    maxFiles: 1,
    multiple: false,
  });

  const remove = () => {
    onChange(null);
    setPreviewURL(null);
  };

  return (
    <div>
      {preview ? (
        <div className="relative overflow-hidden rounded-xl border border-self-strong">
          <img
            src={preview}
            alt="Cover preview"
            className="aspect-video w-full object-cover"
          />
          <button
            type="button"
            onClick={remove}
            className="absolute right-2 top-2 grid size-7 place-items-center rounded-full bg-black/60 text-white backdrop-blur-sm transition-colors hover:bg-black/80"
          >
            <X className="size-4" />
          </button>
        </div>
      ) : (
        <div
          {...getRootProps()}
          className={cn(
            "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-self-strong px-4 py-10 text-center transition-colors hover:border-primary/40 hover:bg-muted/30",
            isDragActive && "border-primary/60 bg-primary/5",
            error && "border-destructive/60 bg-destructive/5",
          )}
        >
          <input {...getInputProps()} />
          {isDragActive ? (
            <Upload className="size-8 text-primary/70" />
          ) : (
            <ImageIcon className="size-8 text-muted-foreground/50" />
          )}
          <div>
            <p className="text-sm font-medium">
              {isDragActive ? "Drop image here" : "Drag & drop a cover image"}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              JPEG, PNG or WebP · up to 5 MB
            </p>
          </div>
        </div>
      )}
      {error && !preview && (
        <p className="mt-1.5 text-sm text-destructive">{error}</p>
      )}
    </div>
  );
}
