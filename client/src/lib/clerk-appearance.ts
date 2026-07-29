import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import type { Appearance } from "@clerk/shared/types";

const baseFont =
  '"Inter", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif';

const darkAppearance: Appearance = {
  variables: {
    colorPrimary: "#fafafa",
    colorText: "#fafafa",
    colorTextSecondary: "#a1a1aa",
    colorBackground: "#111111",
    colorInputBackground: "#171717",
    colorInputText: "#fafafa",
    colorDanger: "#ef4444",
    borderRadius: "8px",
    fontFamily: baseFont,
  },
  elements: {
    rootBox: "mx-auto w-full",
    card: "rounded-xl border border-[#262626] bg-[#111111]! shadow-lg",
    headerTitle: "font-display text-2xl font-semibold tracking-[-0.028em] text-[#fafafa]!",
    headerSubtitle: "text-[0.9375rem] tracking-[-0.01em] text-[#a1a1aa]!",
    socialButtonsBlockButton:
      "border border-[#262626] bg-[#0a0a0a]! text-[#fafafa]! shadow-xs hover:bg-[#171717]",
    formButtonPrimary:
      "bg-[#fafafa]! text-[#0a0a0a]! shadow-sm hover:bg-[#e4e4e7] focus:shadow-[0_0_0_3px_rgba(59,130,246,0.35)]",
    formFieldInput:
      "border-[#262626] bg-[#171717]! text-[#fafafa]! shadow-xs focus:border-[#3b82f6] focus:ring-[3px] focus:ring-[#3b82f6]/30",
    footerActionLink: "text-[#a1a1aa]! hover:text-[#fafafa]!",
    identityPreviewEditButton: "text-[#a1a1aa]! hover:text-[#fafafa]!",
    formFieldAction: "text-[#a1a1aa]! hover:text-[#fafafa]!",
    dividerLine: "bg-[#262626]!",
    dividerText: "text-[#a1a1aa]!",
    userButtonPopoverRootBox: "bg-transparent!",
    userButtonPopoverCard: "rounded-xl border border-[#262626] bg-[#111111]! shadow-xl",
    userButtonPopoverMain: "bg-[#111111]!",
    userButtonPopoverActions: "bg-[#111111]!",
    userButtonPopoverActionButton: "text-[#fafafa]! hover:bg-[#171717]",
    userButtonPopoverActionButton__signOut: "text-[#ef4444]! hover:bg-[#171717]",
    userButtonPopoverActionButtonIconBox: "bg-transparent!",
    userButtonPopoverActionButtonIcon: "text-[#a1a1aa]!",
    userButtonPopoverFooter: "border-t border-[#262626] bg-[#111111]!",
    userButtonPopoverFooterPagesLink: "text-[#a1a1aa]! hover:text-[#fafafa]!",
    userButtonPopoverFooterPagesLink__terms: "text-[#a1a1aa]! hover:text-[#fafafa]!",
    userButtonPopoverFooterPagesLink__privacy: "text-[#a1a1aa]! hover:text-[#fafafa]!",
    userPreview__userButton: "bg-[#111111]!",
    userPreviewAvatarBox__userButton: "bg-transparent!",
    userPreviewTextContainer__userButton: "bg-[#111111]!",
    userPreviewMainIdentifier__userButton: "text-[#fafafa]!",
    userPreviewMainIdentifierText__userButton: "text-[#fafafa]!",
    userPreviewSecondaryIdentifier__userButton: "text-[#a1a1aa]!",
  },
};

const lightAppearance: Appearance = {
  variables: {
    colorPrimary: "#18181b",
    colorText: "#18181b",
    colorTextSecondary: "#71717a",
    colorBackground: "#ffffff",
    colorInputBackground: "#fafafa",
    colorInputText: "#18181b",
    colorDanger: "#dc2626",
    borderRadius: "8px",
    fontFamily: baseFont,
  },
  elements: {
    rootBox: "mx-auto w-full",
    card: "rounded-xl border border-[#e4e4e7] bg-[#ffffff]! shadow-lg",
    headerTitle: "font-display text-2xl font-semibold tracking-[-0.028em] text-[#18181b]!",
    headerSubtitle: "text-[0.9375rem] tracking-[-0.01em] text-[#71717a]!",
    socialButtonsBlockButton:
      "border border-[#e4e4e7] bg-[#fafafa]! text-[#18181b]! shadow-xs hover:bg-[#f4f4f5]",
    formButtonPrimary:
      "bg-[#18181b]! text-[#fafafa]! shadow-sm hover:bg-[#27272a] focus:shadow-[0_0_0_3px_rgba(59,130,246,0.35)]",
    formFieldInput:
      "border-[#e4e4e7] bg-[#fafafa]! text-[#18181b]! shadow-xs focus:border-[#3b82f6] focus:ring-[3px] focus:ring-[#3b82f6]/30",
    footerActionLink: "text-[#71717a]! hover:text-[#18181b]!",
    identityPreviewEditButton: "text-[#71717a]! hover:text-[#18181b]!",
    formFieldAction: "text-[#71717a]! hover:text-[#18181b]!",
    dividerLine: "bg-[#e4e4e7]!",
    dividerText: "text-[#71717a]!",
    userButtonPopoverRootBox: "bg-transparent!",
    userButtonPopoverCard: "rounded-xl border border-[#e4e4e7] bg-[#ffffff]! shadow-xl",
    userButtonPopoverMain: "bg-[#ffffff]!",
    userButtonPopoverActions: "bg-[#ffffff]!",
    userButtonPopoverActionButton: "text-[#18181b]! hover:bg-[#f4f4f5]",
    userButtonPopoverActionButton__signOut: "text-[#dc2626]! hover:bg-[#f4f4f5]",
    userButtonPopoverActionButtonIconBox: "bg-transparent!",
    userButtonPopoverActionButtonIcon: "text-[#71717a]!",
    userButtonPopoverFooter: "border-t border-[#e4e4e7] bg-[#ffffff]!",
    userButtonPopoverFooterPagesLink: "text-[#71717a]! hover:text-[#18181b]!",
    userButtonPopoverFooterPagesLink__terms: "text-[#71717a]! hover:text-[#18181b]!",
    userButtonPopoverFooterPagesLink__privacy: "text-[#71717a]! hover:text-[#18181b]!",
    userPreview__userButton: "bg-[#ffffff]!",
    userPreviewAvatarBox__userButton: "bg-transparent!",
    userPreviewTextContainer__userButton: "bg-[#ffffff]!",
    userPreviewMainIdentifier__userButton: "text-[#18181b]!",
    userPreviewMainIdentifierText__userButton: "text-[#18181b]!",
    userPreviewSecondaryIdentifier__userButton: "text-[#71717a]!",
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
