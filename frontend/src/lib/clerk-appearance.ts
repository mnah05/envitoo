import type { Appearance } from "@clerk/shared/types";

export const clerkAppearance: Appearance = {
  variables: {
    colorPrimary: "#14b8a6",
    colorText: "#f4f4f8",
    colorTextSecondary: "#9b9bb0",
    colorBackground: "#0c0c12",
    colorInputBackground: "#16161f",
    colorInputText: "#f4f4f8",
    colorDanger: "#f43f5e",
    borderRadius: "8px",
    fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", "Helvetica Neue", Inter, ui-sans-serif, system-ui, sans-serif',
  },
  elements: {
    rootBox: "mx-auto w-full",
    card: "rounded-xl border border-[#24243a] bg-[#0c0c12]! shadow-lg shadow-teal-500/10",
    headerTitle: "font-display text-2xl font-semibold tracking-[-0.028em] text-[#f4f4f8]!",
    headerSubtitle: "text-[0.9375rem] tracking-[-0.01em] text-[#9b9bb0]!",
    socialButtonsBlockButton:
      "border border-[#24243a] bg-[#07070b]! text-[#f4f4f8]! shadow-xs hover:bg-[#16161f]",
    formButtonPrimary:
      "bg-[#14b8a6]! text-[#042f2e]! shadow-sm shadow-teal-500/25 hover:bg-[#0d9488] focus:shadow-[0_0_0_3px_rgba(20,184,166,0.35)]",
    formFieldInput:
      "border-[#24243a] bg-[#16161f]! text-[#f4f4f8]! shadow-xs focus:border-[#14b8a6] focus:ring-[3px] focus:ring-[#14b8a6]/30",
    footerActionLink: "text-[#5eead4]! hover:text-[#99f6e4]!",
    identityPreviewEditButton: "text-[#5eead4]! hover:text-[#99f6e4]!",
    formFieldAction: "text-[#5eead4]! hover:text-[#99f6e4]!",
    dividerLine: "bg-[#24243a]!",
    dividerText: "text-[#9b9bb0]!",

    userButtonPopoverRootBox: "bg-transparent!",
    userButtonPopoverCard:
      "rounded-xl border border-[#24243a] bg-[#0c0c12]! shadow-xl",
    userButtonPopoverMain: "bg-[#0c0c12]!",
    userButtonPopoverActions: "bg-[#0c0c12]!",
    userButtonPopoverActionButton: "text-[#f4f4f8]! hover:bg-[#16161f]",
    userButtonPopoverActionButton__signOut:
      "text-[#f43f5e]! hover:bg-[#16161f]",
    userButtonPopoverActionButtonIconBox: "bg-transparent!",
    userButtonPopoverActionButtonIcon: "text-[#9b9bb0]!",
    userButtonPopoverFooter: "border-t border-[#24243a] bg-[#0c0c12]!",
    userButtonPopoverFooterPagesLink:
      "text-[#9b9bb0]! hover:text-[#f4f4f8]!",
    userButtonPopoverFooterPagesLink__terms:
      "text-[#9b9bb0]! hover:text-[#f4f4f8]!",
    userButtonPopoverFooterPagesLink__privacy:
      "text-[#9b9bb0]! hover:text-[#f4f4f8]!",
    userPreview__userButton: "bg-[#0c0c12]!",
    userPreviewAvatarBox__userButton: "bg-transparent!",
    userPreviewTextContainer__userButton: "bg-[#0c0c12]!",
    userPreviewMainIdentifier__userButton: "text-[#f4f4f8]!",
    userPreviewMainIdentifierText__userButton: "text-[#f4f4f8]!",
    userPreviewSecondaryIdentifier__userButton: "text-[#9b9bb0]!",
  },
};
