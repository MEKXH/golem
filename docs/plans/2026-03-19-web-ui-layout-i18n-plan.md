# WebUI Layout and I18n Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the current desktop WebUI layout regressions and add browser-aware bilingual support with manual language switching.

**Architecture:** Keep the existing embedded Vue + Vite app and Gateway embedding model, but add a lightweight frontend locale store plus a minimal Vue unit-test setup. Layout fixes stay in the existing component structure and shared stylesheet so the Go backend contract remains unchanged.

**Tech Stack:** Vue 3, Vue Router, TypeScript, Vite, Vitest, Vue Test Utils, Go embed gateway assets.

---

### Task 1: Document the design and prepare the isolated worktree

**Files:**
- Create: `docs/plans/2026-03-19-web-ui-layout-i18n-design.md`
- Create: `docs/plans/2026-03-19-web-ui-layout-i18n-plan.md`

**Step 1: Verify isolated worktree baseline**

Run: `git worktree list`
Expected: `codex/web-ui-i18n-layout` exists and points at a clean branch from `dev`.

**Step 2: Verify repository baseline**

Run: `go test ./...`
Expected: PASS before touching WebUI logic.

**Step 3: Save design and plan docs**

Write both docs under `docs/plans/`.

**Step 4: Commit later with `-f`**

Because `docs/plans` is ignored in this repo, remember to use `git add -f docs/plans/...` when committing.

### Task 2: Add failing frontend tests for locale behavior

**Files:**
- Modify: `web/package.json`
- Create: `web/vitest.config.ts`
- Create: `web/src/lib/locale.test.ts`
- Create: `web/src/components/console/ConsoleTopbar.test.ts`
- Create: `web/src/test/setup.ts`

**Step 1: Write the failing test for locale detection**

Cover:
- Chinese browser locale resolves to `zh-CN`
- non-Chinese locale resolves to `en`
- persisted override wins over browser locale

**Step 2: Write the failing topbar render test**

Cover:
- translated heading/button text renders from provided locale
- locale toggle renders both language actions

**Step 3: Run the frontend tests to verify RED**

Run: `npm --prefix web run test`
Expected: FAIL because locale utilities/test harness do not exist yet.

**Step 4: Commit after tests are in place**

Commit message: `test: add web ui locale coverage`

### Task 3: Implement lightweight locale store and translation dictionaries

**Files:**
- Create: `web/src/lib/locale.ts`
- Create: `web/src/lib/messages.ts`
- Modify: `web/src/main.ts`
- Modify: `web/src/App.vue`
- Modify: `web/src/types.ts` if locale types belong there

**Step 1: Add minimal locale types and dictionaries**

Support:
- `en`
- `zh-CN`

**Step 2: Implement browser detection and local override**

Behavior:
- read override from `localStorage`
- else resolve from browser languages
- treat all `zh*` browser tags as `zh-CN`
- fallback to `en`

**Step 3: Expose a small composable/helper**

Needs:
- current locale
- `setLocale()`
- translation lookup helper

**Step 4: Run targeted tests to verify GREEN**

Run: `npm --prefix web run test -- locale`
Expected: PASS.

**Step 5: Commit**

Commit message: `feat: add web ui locale store`

### Task 4: Wire bilingual text and language switch into landing and console UI

**Files:**
- Modify: `web/src/components/landing/HeroSection.vue`
- Modify: `web/src/components/landing/CapabilityBands.vue`
- Modify: `web/src/components/landing/GeoEvolutionSection.vue`
- Modify: `web/src/components/landing/ConsoleCTA.vue`
- Modify: `web/src/components/console/ConsoleTopbar.vue`
- Modify: `web/src/components/console/ConnectionPanel.vue`
- Modify: `web/src/components/console/ComposerPanel.vue`
- Modify: `web/src/components/console/ChatTimeline.vue`
- Modify: `web/src/pages/ConsolePage.vue`

**Step 1: Replace hard-coded user-facing strings with translated lookups**

Keep route behavior and props intact.

**Step 2: Add manual language toggle UI**

Put it in the console topbar and make it globally reflected across routes.

**Step 3: Re-run component test suite**

Run: `npm --prefix web run test`
Expected: PASS.

**Step 4: Commit**

Commit message: `feat: localize embedded web ui`

### Task 5: Fix desktop landing and console layout regressions

**Files:**
- Modify: `web/src/styles.css`
- Modify: `web/src/components/console/ConsoleTopbar.vue` if structural wrappers are needed
- Modify: `web/src/pages/ConsolePage.vue` if panel wrappers need stronger layout hooks
- Modify: `web/src/components/landing/HeroSection.vue` if the right-side wrapper needs a dedicated class

**Step 1: Write the smallest markup changes needed for stable layout hooks**

Add named wrappers only where CSS needs clearer targets.

**Step 2: Adjust desktop hero alignment**

Expected outcome:
- right card stack sits farther right
- left copy remains readable and not crowded

**Step 3: Adjust console equal-height layout**

Expected outcome:
- left rail fills row height
- right main card fills row height
- timeline expands, composer stays docked

**Step 4: Adjust topbar desktop row alignment**

Expected outcome:
- title block left
- status, locale switch, refresh, and back action aligned in one row until breakpoint collapse

**Step 5: Manual desktop verification**

Check `/` and `/console` in browser width around 1280–1440px.

**Step 6: Commit**

Commit message: `fix: refine web ui desktop layout`

### Task 6: Rebuild embedded assets and run full verification

**Files:**
- Modify: `internal/gateway/webui/**` (generated by sync script)
- Verify: `internal/gateway/server.go`
- Verify: `internal/gateway/server_test.go`

**Step 1: Run frontend test suite**

Run: `npm --prefix web run test`
Expected: PASS.

**Step 2: Run frontend typecheck**

Run: `npm --prefix web run typecheck`
Expected: PASS.

**Step 3: Rebuild and sync embedded assets**

Run: `npm --prefix web run build:gateway`
Expected: PASS and updated files under `internal/gateway/webui/`.

**Step 4: Run full Go verification**

Run: `go test ./...`
Expected: PASS.

**Step 5: Final commit**

Commit message: `fix: add bilingual web ui polish`
