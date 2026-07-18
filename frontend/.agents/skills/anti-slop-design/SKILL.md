---
name: anti-slop-design
description: Use when building, modifying, or reviewing any frontend UI in this project. Enforces the pols.dev anti-slop design law - a comprehensive checklist of design tells, layout pitfalls, and premium craft principles to avoid generic AI output and ship considered, distinctive interfaces.
---

# Anti-Slop Design Law

The full anti-slop design law is loaded as global instructions from `~/.config/opencode/AGENTS.md`. Read it there before any frontend work. This skill exists to remind you it applies to THIS project too.

## When to apply

Every time you create or modify any React component, page layout, CSS, or visual element in `frontend/src/`.

## This project's design context

- **Stack**: React 19, Vite, Tailwind CSS v4, shadcn/ui (new-york style), Framer Motion, Lucide React
- **Dark mode only**, near-black base `#07070b`
- **Teal primary** `#14b8a6` with cyan, rose, amber, emerald accents
- **Typography**: Apple-first (SF Pro), Inter/Inter Tight elsewhere, Inter Tight for display
- **Responsive**: mobile bottom nav + desktop header

## Key rules to remember for this project

1. **Lucide React** is already used throughout - this is fine, it's the project's chosen icon set
2. **Inter/Inter Tight** are the project fonts - already established, do not swap
3. **No blue-purple gradients** - the palette is teal/cyan based, keep it that way
4. **No generic hero stacks** - compose pages as whole compositions from the brief
5. **Content visible by default** - never gate content on animations completing
6. **Clear the cut** - check clip-path, overflow:hidden, fixed heights for content clipping
7. **Center what you mean to center** - verify mathematically and optically
8. **No hover boop on buttons** - tonal color shifts instead
9. **No growing underline animations** - change state cleanly
10. **No sun-moon theme toggle** - the project uses `next-themes` with dark-only default

## Before shipping any UI change

Walk the full anti-slop checklist from AGENTS.md point by point. Fix what falls short before calling it done. The user's explicit instructions override these defaults, but absent direction, this file is law.
