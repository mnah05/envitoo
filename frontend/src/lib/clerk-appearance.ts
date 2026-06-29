import type { Appearance } from "@clerk/shared/types";

/** Shared Clerk theme aligned with Pulse design tokens (index.css). */
export const clerkAppearance: Appearance = {
  variables: {
    colorPrimary: "#ff385c",
    colorText: "#222222",
    colorTextSecondary: "#717171",
    colorBackground: "#ffffff",
    colorInputBackground: "#ffffff",
    colorInputText: "#222222",
    colorDanger: "#c1351f",
    borderRadius: "14px",
    fontFamily: '"Inter", ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
  },
  elements: {
    rootBox: "mx-auto w-full",
    card: "rounded-2xl border border-border bg-card shadow-sm",
    headerTitle: "text-2xl font-bold tracking-tight text-foreground",
    headerSubtitle: "text-muted-foreground",
    socialButtonsBlockButton:
      "border border-border bg-background text-foreground shadow-xs hover:bg-accent",
    formButtonPrimary:
      "bg-primary text-primary-foreground shadow-xs hover:bg-primary/90 focus:shadow-[0_0_0_3px_color-mix(in_srgb,var(--ring)_50%,transparent)]",
    formFieldInput:
      "border-input bg-background text-foreground shadow-xs focus:border-ring focus:ring-[3px] focus:ring-ring/50",
    footerActionLink: "text-primary hover:text-primary/90",
    identityPreviewEditButton: "text-primary hover:text-primary/90",
    formFieldAction: "text-primary hover:text-primary/90",
    dividerLine: "bg-border",
    dividerText: "text-muted-foreground",
  },
};