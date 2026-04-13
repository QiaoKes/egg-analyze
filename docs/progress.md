# Egg Analyze Progress

## Goal

Build a cross-platform desktop tool in Go that:

- captures a screenshot through a floating launcher window
- extracts egg size and weight with OCR
- uses the live rocom site data source
- computes egg candidate probabilities locally
- provides a GUI plus floating launcher/result windows

## Implementation Plan

1. Create project skeleton, config storage, and progress log
2. Pull and cache `https://rocom.mfsky.qzz.io/data/egg-measurements-final.json`
3. Port the site matching algorithm into Go
4. Integrate RapidOCR
5. Build screenshot analysis flow for flameshot region captures and local image imports
6. Add Fyne GUI and floating launcher/result workflow
7. Validate against the `picture` samples and document remaining gaps

## Confirmed Facts

- The target site is a static front-end app, not a query backend service.
- The public data endpoint for egg matching is `https://rocom.mfsky.qzz.io/data/egg-measurements-final.json`.
- The site computes probabilities in the browser from the dataset.
- Flameshot officially supports `flameshot gui`, `-p/--path`, and `-s/--accept-on-select`, which is enough for external region capture integration.
- Flameshot is available on Linux, Windows, and macOS via the official project documentation.

## Current Blockers / Risks

1. Site “API” is actually public JSON plus front-end scoring logic, so the desktop app must replicate the scoring algorithm locally.
2. OCR still has visible noise on low-resolution numbers, so extraction depends on numeric normalization and pairing instead of fixed text anchors.
3. The application now depends on an external screenshot tool (`flameshot`) being installed on the host.
4. Fyne public API does not expose a true always-on-top or freely draggable floating ball, so the app currently uses borderless utility windows.
5. Release packaging now aims to bundle both `flameshot` and a Python runtime for RapidOCR, which increases artifact size.

## Work Log

- 2026-04-12: inspected sample screenshots in `picture/`
- 2026-04-12: verified the site serves public JSON with permissive CORS
- 2026-04-12: confirmed the reference repo is front-end only and contains the scoring logic
- 2026-04-12: replaced Windows OCR with RapidOCR and validated against local samples
- 2026-04-12: switched screenshot capture to external `flameshot` integration
- 2026-04-12: added bundled-flameshot lookup and GitHub Actions packaging for Windows/macOS artifacts
- 2026-04-12: added bundled-Python lookup and package-time Python runtime installation for OCR
- 2026-04-12: added sample-based analyzer tests and maintained passing `go test ./...`
