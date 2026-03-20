# WebUI Layout and I18n Design

**Date:** 2026-03-19
**Branch:** `codex/web-ui-i18n-layout`

## Problem

The first embedded Vue WebUI works functionally, but desktop presentation has three concrete UX regressions:

- The landing-page hero card stack sits too close to center instead of feeling anchored to the right-hand stage area.
- The console sidebar and main content area do not feel visually aligned in height, which makes the operator layout look unfinished.
- The console topbar title and action controls break alignment, so status and primary actions do not reliably read as one row on desktop.

A second gap is language support: the UI is English-only and has no browser-aware locale detection or user-controlled language override.

## Goals

- Fix the desktop landing-page right-side composition so the three cards feel intentionally right-aligned.
- Make the console desktop layout feel balanced, with the left rail visually matching the right content column.
- Keep console topbar controls on one coherent desktop row whenever width allows.
- Add browser-aware bilingual support for English and Simplified Chinese.
- Allow manual language switching that overrides automatic browser detection and persists locally.

## Non-Goals

- No backend API changes.
- No server-side locale negotiation.
- No third-party full i18n framework in this iteration.
- No mobile-specific redesign beyond preserving current responsiveness.

## Approach Options

### Option 1: CSS-only patch plus ad-hoc text toggles

Adjust current classes and branch on a `lang` flag inside components.

Pros:
- Fastest initial patch.

Cons:
- Text duplication spreads through components.
- Hard to scale as the WebUI grows.
- Browser detection, manual override, and shared labels become brittle.

### Option 2: Lightweight local i18n layer plus layout refactor

Introduce a small locale store/composable with message dictionaries, browser detection, and local override. Refine component structure and CSS tokens for the landing hero, console grid, and topbar.

Pros:
- Solves current issue cleanly.
- Keeps text centralized.
- Stays small and framework-native.
- Easy to extend to more routes/components.

Cons:
- Slightly more code movement than a CSS-only patch.

### Option 3: Full i18n plugin integration

Install and wire a full i18n ecosystem package now.

Pros:
- Standardized ecosystem solution.

Cons:
- Too much scope for two locales and one small app.
- Adds overhead before the UI vocabulary stabilizes.

## Chosen Design

Use Option 2.

### Information Architecture

Keep the two-page structure unchanged:

- `/` remains the marketing landing page.
- `/console` remains the operator console.

Add one shared locale state with these rules:

- On first load, derive locale from `navigator.languages` / `navigator.language`.
- Normalize any Chinese-family browser locale (`zh`, `zh-CN`, `zh-HK`, `zh-TW`, etc.) to the Simplified Chinese UI for now.
- Default all other locales to English.
- If the user manually changes language, persist the override in `localStorage` and prefer it on later visits.

### Frontend Structure

Add a tiny i18n layer in `web/src/lib/` or `web/src/composables/`:

- locale type definition
- locale messages object
- locale resolver (`detectPreferredLocale`)
- small shared state/composable (`useLocale`)
- translation helper (`t(key)` or mapped sections)

This layer should be framework-light and not depend on server configuration.

### Desktop Layout Fixes

Landing page:

- Keep a two-column hero grid.
- Change the right column to justify itself toward the right edge of the max-width container.
- Increase the perceived stage offset of the three cards using a dedicated wrapper width and right alignment rather than only transform hacks.

Console page:

- Keep the two-column console layout.
- Make the left rail stretch with the row and make the card itself fill available height.
- Keep the right card as a flex/grid column so the timeline expands and the composer remains docked at the bottom.

Console topbar:

- Split topbar content into a title block and a desktop action rail.
- Use explicit alignment and spacing so status pills, locale toggle, refresh button, and back link stay on one line until the responsive breakpoint is hit.

### Component Changes

- Introduce a reusable locale switch control in the top-level shell or topbar.
- Update landing and console components to read all user-facing copy from the locale dictionary.
- Keep route structure unchanged.

### Testing Strategy

Use TDD for deterministic behavior:

- Add frontend unit tests for locale detection and locale override behavior.
- Add component-level rendering tests to confirm locale toggle labels and translated headings change with locale state.
- Use manual visual verification for CSS layout fixes on desktop because equal-height and alignment issues are visual, not semantic.
- Rebuild embedded assets and run `go test ./...` to confirm gateway embedding stays healthy.

## Risks

- Because the current WebUI has no frontend test runner, adding a minimal Vue test setup is part of the change.
- Browser-language normalization for Chinese variants is intentionally simplified; future Traditional Chinese support will need a more granular mapping.
- Visual balance on desktop depends on testing against the current max-width and card sizes after translation length changes.

## Acceptance Criteria

- Landing page right-side hero cards are visually shifted and anchored farther right on desktop.
- Console left rail and right main panel present as a balanced desktop layout.
- Console topbar controls appear on one row on desktop.
- WebUI chooses English or Chinese automatically from browser language.
- Users can manually switch language and the selection persists locally.
- `npm --prefix web run test`, `npm --prefix web run typecheck`, `npm --prefix web run build:gateway`, and `go test ./...` pass.
