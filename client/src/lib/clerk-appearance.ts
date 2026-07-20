import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import type { Appearance } from "@clerk/shared/types";

const baseFont = '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", "Helvetica Neue", "Satoshi", "Inter", ui-sans-serif, system-ui, sans-serif';

const darkAppearance: Appearance = {
  variables: {
    colorPrimary: "#14b8a6",
    colorText: "#f4f4f8",
    colorTextSecondary: "#9b9bb0",
    colorBackground: "#0c0c12",
    colorInputBackground: "#16161f",
    colorInputText: "#f4f4f8",
    colorDanger: "#f43f5e",
    borderRadius: "8px",
    fontFamily: baseFont,
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
    userButtonPopoverCard: "rounded-xl border border-[#24243a] bg-[#0c0c12]! shadow-xl",
    userButtonPopoverMain: "bg-[#0c0c12]!",
    userButtonPopoverActions: "bg-[#0c0c12]!",
    userButtonPopoverActionButton: "text-[#f4f4f8]! hover:bg-[#16161f]",
    userButtonPopoverActionButton__signOut: "text-[#f43f5e]! hover:bg-[#16161f]",
    userButtonPopoverActionButtonIconBox: "bg-transparent!",
    userButtonPopoverActionButtonIcon: "text-[#9b9bb0]!",
    userButtonPopoverFooter: "border-t border-[#24243a] bg-[#0c0c12]!",
    userButtonPopoverFooterPagesLink: "text-[#9b9bb0]! hover:text-[#f4f4f8]!",
    userButtonPopoverFooterPagesLink__terms: "text-[#9b9bb0]! hover:text-[#f4f4f8]!",
    userButtonPopoverFooterPagesLink__privacy: "text-[#9b9bb0]! hover:text-[#f4f4f8]!",
    userPreview__userButton: "bg-[#0c0c12]!",
    userPreviewAvatarBox__userButton: "bg-transparent!",
    userPreviewTextContainer__userButton: "bg-[#0c0c12]!",
    userPreviewMainIdentifier__userButton: "text-[#f4f4f8]!",
    userPreviewMainIdentifierText__userButton: "text-[#f4f4f8]!",
    userPreviewSecondaryIdentifier__userButton: "text-[#9b9bb0]!",
  },
};

const lightAppearance: Appearance = {
  variables: {
    colorPrimary: "#0d9488",
    colorText: "#1a1a2e",
    colorTextSecondary: "#6b6b80",
    colorBackground: "#ffffff",
    colorInputBackground: "#f8f8fb",
    colorInputText: "#1a1a2e",
    colorDanger: "#e11d48",
    borderRadius: "8px",
    fontFamily: baseFont,
  },
  elements: {
    rootBox: "mx-auto w-full",
    card: "rounded-xl border border-[#e2e2ec] bg-[#ffffff]! shadow-lg",
    headerTitle: "font-display text-2xl font-semibold tracking-[-0.028em] text-[#1a1a2e]!",
    headerSubtitle: "text-[0.9375rem] tracking-[-0.01em] text-[#6b6b80]!",
    socialButtonsBlockButton:
      "border border-[#e2e2ec] bg-[#f8f8fb]! text-[#1a1a2e]! shadow-xs hover:bg-[#f0f0f5]",
    formButtonPrimary:
      "bg-[#0d9488]! text-[#ffffff]! shadow-sm hover:bg-[#0f766e] focus:shadow-[0_0_0_3px_rgba(13,148,136,0.35)]",
    formFieldInput:
      "border-[#e2e2ec] bg-[#f8f8fb]! text-[#1a1a2e]! shadow-xs focus:border-[#0d9488] focus:ring-[3px] focus:ring-[#0d9488]/30",
    footerActionLink: "text-[#0d9488]! hover:text-[#0f766e]!",
    identityPreviewEditButton: "text-[#0d9488]! hover:text-[#0f766e]!",
    formFieldAction: "text-[#0d9488]! hover:text-[#0f766e]!",
    dividerLine: "bg-[#e2e2ec]!",
    dividerText: "text-[#6b6b80]!",
    userButtonPopoverRootBox: "bg-transparent!",
    userButtonPopoverCard: "rounded-xl border border-[#e2e2ec] bg-[#ffffff]! shadow-xl",
    userButtonPopoverMain: "bg-[#ffffff]!",
    userButtonPopoverActions: "bg-[#ffffff]!",
    userButtonPopoverActionButton: "text-[#1a1a2e]! hover:bg-[#f0f0f5]",
    userButtonPopoverActionButton__signOut: "text-[#e11d48]! hover:bg-[#f0f0f5]",
    userButtonPopoverActionButtonIconBox: "bg-transparent!",
    userButtonPopoverActionButtonIcon: "text-[#6b6b80]!",
    userButtonPopoverFooter: "border-t border-[#e2e2ec] bg-[#ffffff]!",
    userButtonPopoverFooterPagesLink: "text-[#6b6b80]! hover:text-[#1a1a2e]!",
    userButtonPopoverFooterPagesLink__terms: "text-[#6b6b80]! hover:text-[#1a1a2e]!",
    userButtonPopoverFooterPagesLink__privacy: "text-[#6b6b80]! hover:text-[#1a1a2e]!",
    userPreview__userButton: "bg-[#ffffff]!",
    userPreviewAvatarBox__userButton: "bg-transparent!",
    userPreviewTextContainer__userButton: "bg-[#ffffff]!",
    userPreviewMainIdentifier__userButton: "text-[#1a1a2e]!",
    userPreviewMainIdentifierText__userButton: "text-[#1a1a2e]!",
    userPreviewSecondaryIdentifier__userButton: "text-[#6b6b80]!",
  },
};

export function useClerkAppearance(): Appearance {
  const { resolvedTheme } = useTheme();
  const [appearance, setAppearance] = useState<Appearance>(darkAppearance);

  useEffect(() => {
    setAppearance(resolvedTheme === "dark" ? darkAppearance : lightAppearance);
  }, [resolvedTheme]);

  return appearance;
}
