---
name: anti-slop-design
description: Use when building, modifying, or reviewing any frontend UI in this project. Enforces the pols.dev anti-slop design law - a comprehensive checklist of design tells, layout pitfalls, and premium craft principles to avoid generic AI output and ship considered, distinctive interfaces.
---

# Anti-Slop Design Law

The full anti-slop design law is loaded as global instructions from `~/.config/opencode/AGENTS.md`. Read it there before any frontend work. This skill exists to remind you it applies to THIS project too.

## When to apply

Every time you create or modify any React component, page layout, CSS, or visual element in `client/src/`.

## This project's design context

- **Stack**: React 19, Vite, Tailwind CSS v4, shadcn/ui (new-york style), Framer Motion / Motion, Lucide React + Lucide Animated
- **Theme**: Clean zinc neutrals, monochrome primary buttons, blue focus ring only; light + dark
- **Typography**: Inter (body/UI) + Inter Tight (display/headings)
- **Icons**: Lucide for dense chrome (nav, forms, tables). Lucide Animated (`lucide-animated`) for marketing features, how-it-works, empty states, and large dashboard stats only
- **Responsive**: mobile bottom nav + desktop header

## Key rules to remember for this project

1. **Lucide React** for nav/forms/dense UI — do not animate every icon
2. **Inter + Inter Tight** are the project fonts — do not reintroduce Clash/Satoshi/Sentient or Fontshare
3. **Stay monochrome-first** — zinc UI; color only for focus ring, destructive, and light category cues
4. **No generic hero stacks** — compose pages as whole compositions from the brief
5. **Content visible by default** — never gate content on animations completing
6. **Clear the cut** — check clip-path, overflow:hidden, fixed heights for content clipping
7. **Center what you mean to center** — verify mathematically and optically
8. **No hover boop on buttons** — tonal color shifts instead
9. **No growing underline animations** — change state cleanly
10. **Quiet product surfaces** — soft borders; ambient washes only on Landing hero

## Before shipping any UI change

Walk the full anti-slop checklist from AGENTS.md point by point. Fix what falls short before calling it done. The user's explicit instructions override these defaults, but absent direction, this file is law.
