# Egg Analyze Progress

## Goal

Build a Windows desktop tool in Go that:

- captures a screenshot through a configurable global hotkey
- extracts egg size and weight with OCR
- uses the live rocom site data source
- computes egg candidate probabilities locally
- provides a GUI and minimizes to the Windows tray

## Implementation Plan

1. Create project skeleton, config storage, and progress log
2. Pull and cache `https://rocom.mfsky.qzz.io/data/egg-measurements-final.json`
3. Port the site matching algorithm into Go
4. Integrate Windows built-in OCR through PowerShell WinRT
5. Build screenshot analysis flow for full-screen captures and local image imports
6. Add Fyne GUI, tray actions, and configurable hotkey
7. Validate against the `picture` samples and document remaining gaps

## Confirmed Facts

- The target site is a static front-end app, not a query backend service.
- The public data endpoint for egg matching is `https://rocom.mfsky.qzz.io/data/egg-measurements-final.json`.
- The site computes probabilities in the browser from the dataset.
- Windows built-in OCR is available through Windows PowerShell and works in the current environment.

## Current Blockers / Risks

1. Site “API” is actually public JSON plus front-end scoring logic, so the desktop app must replicate the scoring algorithm locally.
2. Whole-image OCR has visible noise on Chinese UI text; the current implementation mitigates this with egg-title anchors and numeric cleanup, but some multi-egg screenshots may still miss low-quality entries.
3. First iteration targets primary-screen capture. Multi-monitor region selection is intentionally deferred to reduce failure risk.
4. GUI framework was switched from Fyne to Windows-native `walk` because the current environment does not provide `gcc`, and Fyne's desktop driver required cgo/OpenGL here.

## Work Log

- 2026-04-12: inspected sample screenshots in `picture/`
- 2026-04-12: verified the site serves public JSON with permissive CORS
- 2026-04-12: confirmed the reference repo is front-end only and contains the scoring logic
- 2026-04-12: verified Windows built-in OCR can read the sample screenshots
- 2026-04-12: implemented Go desktop app skeleton with Windows tray, configurable hotkey, screenshot capture, OCR, and probability analysis
- 2026-04-12: added sample-based analyzer tests and passed `蛋详细页面.png` and `孵蛋页面3.png`
