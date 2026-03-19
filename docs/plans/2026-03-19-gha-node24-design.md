# GitHub Actions Node 24 Upgrade Design

## Goal

Remove GitHub Actions Node.js 20 deprecation warnings from CI and release workflows by upgrading action versions where Node 24-native majors exist, and by explicitly opting the remaining release action path into Node 24.

## Selected Approach

Upgrade the existing workflows in place:

- `actions/checkout@v4` -> `actions/checkout@v5`
- `actions/setup-go@v5` -> `actions/setup-go@v6`
- `actions/upload-artifact@v4` -> `actions/upload-artifact@v6`
- `actions/download-artifact@v4` -> `actions/download-artifact@v8`
- keep `softprops/action-gh-release@v2`, but set `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true` in the release workflow

## Why This Approach

- It removes the current Node 20 deprecation warnings from the official GitHub actions used in `ci.yml` and `release.yml`.
- It keeps the release behavior intact instead of replacing the release action during the same change.
- It minimizes behavioral risk while still opting into the June 2, 2026 Node 24 runtime path now.

## Constraints

- Newer official action majors require newer runner versions. GitHub-hosted runners already satisfy this; self-hosted runners must be at least `2.327.1`, and `actions/checkout@v5` notes `2.329.0` for one credential persistence scenario.
- The repository currently uses GitHub-hosted runners in these workflows, so no workflow-level runner image change is required.

## Verification

- Validate the updated workflow YAML locally for obvious mistakes.
- Run the repository Go test suite to verify no incidental repository state issues before commit.
- The final runtime verification remains the next GitHub Actions run, which should stop emitting the Node 20 deprecation warnings for the upgraded steps.
