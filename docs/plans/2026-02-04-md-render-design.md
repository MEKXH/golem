# TUI Markdown Rendering for Think Blocks

## Goal
Render Markdown for both main assistant content and <think> content in the TUI chat view.

## Context
Currently, Markdown is only rendered when no <think> block is present. When <think> exists, both sections are concatenated without Markdown rendering. This causes raw Markdown to appear in the UI.

## Approach
- Parse the response into think and main segments using the existing <think> regex.
- Render each segment separately via the Glamour renderer.
- Apply the existing visual separation (thinking label + indentation) after rendering.
- Keep the current error behavior: if rendering fails, fall back to raw text and append a render error note.

## Data Flow
1. Receive model response (string)
2. Split into think/main (if <think> present)
3. Render each segment with the renderer
4. Build view content and append to history

## Testing
Introduce unit tests around a small helper that renders segments using a renderer interface. Use a fake renderer to assert both segments are rendered when <think> exists, and only the main segment is rendered otherwise.

## Risks
Minimal; changes are confined to TUI rendering.
