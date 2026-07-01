import type { Appearance } from "@clerk/shared/types";

export const clerkAppearance: Appearance = {
  variables: {
    colorPrimary: "#f5c518",
    colorText: "#e4e4f0",
    colorTextSecondary: "#6e6e8a",
    colorBackground: "#0f0f1e",
    colorInputBackground: "#1a1a35",
    colorInputText: "#e4e4f0",
    colorDanger: "#ff2255",
    borderRadius: "10px",
    fontFamily: '"Inter", ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
  },
  elements: {
    rootBox: "mx-auto w-full",
    // `!` (important) suffix needed on text/bg colors: Clerk's runtime CSS
    // sets these with higher specificity than Tailwind utilities, leaving
    // text black on the dark background without it.
    card: "rounded-2xl border border-[#252545] bg-[#0f0f1e]! shadow-lg shadow-yellow-500/5",
    headerTitle: "text-2xl font-bold tracking-tight text-[#e4e4f0]!",
    headerSubtitle: "text-[#6e6e8a]!",
    socialButtonsBlockButton:
      "border border-[#252545] bg-[#07070d]! text-[#e4e4f0]! shadow-xs hover:bg-[#1a1a30]",
    formButtonPrimary:
      "bg-[#f5c518]! text-[#07070d]! shadow-sm shadow-yellow-500/20 hover:bg-[#f5c518]/90 focus:shadow-[0_0_0_3px_rgba(245,197,24,0.3)]",
    formFieldInput:
      "border-[#252545] bg-[#1a1a35]! text-[#e4e4f0]! shadow-xs focus:border-[#f5c518] focus:ring-[3px] focus:ring-[#f5c518]/30",
    footerActionLink: "text-[#f5c518]! hover:text-[#f5c518]/80",
    identityPreviewEditButton: "text-[#f5c518]! hover:text-[#f5c518]/80",
    formFieldAction: "text-[#f5c518]! hover:text-[#f5c518]/80",
    dividerLine: "bg-[#252545]!",
    dividerText: "text-[#6e6e8a]!",

    // --- UserButton popover (dark mode) ---
    // `!` (important) suffix needed on text colors: Clerk's runtime CSS sets
    // these with higher specificity than Tailwind utilities, leaving text
    // black on the dark background without it.
    userButtonPopoverRootBox: "bg-transparent!",
    userButtonPopoverCard:
      "rounded-xl border border-[#252545] bg-[#0f0f1e]! shadow-xl",
    userButtonPopoverMain: "bg-[#0f0f1e]!",
    userButtonPopoverActions: "bg-[#0f0f1e]!",
    userButtonPopoverActionButton: "text-[#e4e4f0]! hover:bg-[#1a1a30]",
    userButtonPopoverActionButton__signOut:
      "text-[#ff2255]! hover:bg-[#1a1a30]",
    userButtonPopoverActionButtonIconBox: "bg-transparent!",
    userButtonPopoverActionButtonIcon: "text-[#6e6e8a]!",
    userButtonPopoverFooter: "border-t border-[#252545] bg-[#0f0f1e]!",
    userButtonPopoverFooterPagesLink:
      "text-[#6e6e8a]! hover:text-[#e4e4f0]!",
    userButtonPopoverFooterPagesLink__terms:
      "text-[#6e6e8a]! hover:text-[#e4e4f0]!",
    userButtonPopoverFooterPagesLink__privacy:
      "text-[#6e6e8a]! hover:text-[#e4e4f0]!",
    userPreview__userButton: "bg-[#0f0f1e]!",
    userPreviewAvatarBox__userButton: "bg-transparent!",
    userPreviewTextContainer__userButton: "bg-[#0f0f1e]!",
    userPreviewMainIdentifier__userButton: "text-[#e4e4f0]!",
    userPreviewMainIdentifierText__userButton: "text-[#e4e4f0]!",
    userPreviewSecondaryIdentifier__userButton: "text-[#6e6e8a]!",
  },
};
