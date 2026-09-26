# IndigiSnap — Master Context Prompt

Paste this entire file at the start of a new chat to bring a fresh
assistant up to speed on this project.

---

## Project Overview

IndigiSnap is a personal, single-user photo and video gallery app for
Android. It started as a Flask web app running in Termux, launched by
Tasker, viewed in a mobile browser. It was fragile. The current goal is a
real, standalone Android APK.

**Architecture:**
- Android shell (Kotlin): MainActivity, CameraActivity, CameraController,
  ServerService, ServerBridge, SessionState, SessionShot
- Go backend: compiled as libindigisnap.so (c-shared), runs inside the
  app process
- UI (HTML/CSS/JS): served by the Go backend, rendered in a WebView
- Camera: CameraX for photos (multi-shot sessions, inbox pattern),
  system camera intent for single-shot video
- Storage: currently app-private (filesDir/IndigiSnap). SAF folder
  picker is queued.
- Server: binds 127.0.0.1 only

**Current target device:** Samsung Galaxy S25 Ultra, Android 15, SDK 35.

---

## Current State (as of end of 2026-09-26)

**What works:**
- APK builds, installs, launches
- Core browse/view/favorites/batch operations work on the WebView
- CameraX opens, multi-shot capture works, thumbnails populate
- Commit to folder works
- EXIF-aware thumbnail loading added (verification pending)
- Rotate, delete, favorite, share functional (with caveats below)

**What doesn't work or is queued:**
- Camera preview renders landscape despite manifest portrait lock
- Camera review thumbnails sideways (fix added, not verified)
- Browse-page viewer not truly fullscreen
- Browse-page viewer image change lag
- Per-tile action circles are cluttered
- Share button shares URL, not file
- Rotate lacks write verification (data integrity risk)
- Batch delete is irreversible (asymmetry with individual delete)
- No resolution/quality presets
- No tap-to-focus
- No swipe-up camera toggle
- No SAF folder picker (photos invisible to other apps)

See TODO.md for the full queued list.

---

## Development Environment

**Primary dev:** GitHub Codespaces (browser-based)
**Backup dev:** Steam Deck konsole at /home/deck/IndigiSnap (Arch Linux)
**Build:** Local `./gradlew assembleDebug` on Steam Deck
**Install:** ADB over USB

**Steam Deck specifics:**
- JAVA_HOME=/usr/lib/jvm/java-17-openjdk (set in ~/.bashrc)
- ANDROID_HOME=/home/deck/android-sdk
- Port 8090 for local Go server (8080 is taken by Steam webhelper)
- Helper functions in ~/.bashrc: cb (copy command output to clipboard),
  killindigi (kill server processes)

**Working rules:**
- Terminal only, via Python heredocs written to /tmp/*.py files. Never
  long inline heredocs (they break on ! and tab escaping).
- File / Find / Replace for all edits when possible.
- One change at a time, verify after each.
- No deferrals for effort-avoidance. If it's in scope, build it.
- Do not resolve open decisions silently. Ask.
- Do not introduce scope creep. When in doubt, plan first.

---

## Decisions Locked

1. Camera approach: CameraX for photos, system intent for video
2. File naming: {leaf_folder}_{MM-DD-YY}.jpg with _1, _2 dupes
3. Multi-snap: single CameraActivity, thumbnail strip, select-to-delete,
   commit moves files from inbox to target folder
4. Rotate: rewrites file on disk with EXIF reset to NORMAL
5. Delete model: batch delete marks (like individual), doesn't remove
   files immediately. Only commit or discard actually deletes.
6. Color roles:
   - Magenta #FF00FF: theme/borders/icons
   - Green #00FF99: counters/accents/glows
   - Amber #FFB000: cooldown/busy
   - Red #FF3355: delete/destructive
   - BG #07040D, surface #110822
7. Preset names: Highest Quality, Balanced (default), Storage Saver,
   Fast Capture
8. Overflow menu for presets, not top bar
9. Viewer long-press: closes viewer, enters grid select mode with that
   item selected
10. Batch favorite: if all selected favorited → unfavorite all; else
    favorite all

---

## Working Rules For Assistants

- No heredocs longer than 20 lines inline. Write to /tmp/*.py and
  execute.
- When writing patches, match by function signature and closing brace,
  not by exact emoji/escape sequences.
- One build per pass, verify, iterate.
- Correctness (data integrity, memory) before styling.
- When a function's contract changes (e.g., deleteShots), grep all call
  sites first.
- Anything that writes to disk gets write verification (exists, size > 0,
  decodable).
- Sampled decode (inSampleSize) for any thumbnail generation.

---

## Repo

github.com/calvenwilliams1-pixel/IndigiSnap
Package: com.indigisnap.app
Version: 0.1.0 (dev)
minSdk: 24, targetSdk: 34, compileSdk: 34
