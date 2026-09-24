# IndigiSnap — Project Context

Read this first if you are an AI assistant picking up this project cold.

## What This Is

IndigiSnap is a personal, single-user photo and video gallery app for Android. It started as a Python/Flask app running in Termux, launched by Tasker, viewed in a mobile browser. It was fragile. The current goal is a real standalone Android APK.

**Hybrid architecture:**
- Android shell (Kotlin): MainActivity, ServerService, ServerBridge
- Backend (Go): compiled as libindigisnap.so (c-shared), runs inside the app process
- UI (HTML/CSS/JS): served by the Go backend, rendered in a WebView
- Storage: app-private (filesDir/IndigiSnap/)
- Server: binds 127.0.0.1:8080 only (loopback)

Tap the icon -> app launches -> starts embedded Go server -> loads gallery UI in WebView.

## Current Status

- APK builds, installs, launches
- Real UI renders on phone
- Core browse/view/favorites/batch operations work
- Camera buttons exist but do not yet open a real camera session
- Multi-snap capture not yet implemented
- Distribution deferred (personal use only, Play Store later)

See TODO.md for live status. See PRODUCTION_PLAN.md for the roadmap.

## Development Constraints

- Primary dev: GitHub Codespaces (exhausted this month)
- Backup dev: Steam Deck konsole at /home/deck/IndigiSnap (Arch Linux)
- Build: GitHub Actions (auto on push)
- Install: ADB over USB
- Editor: terminal via Python heredocs (python3 << 'PYEOF'). Never cat << EOF - corrupts on Codespaces web terminal.
- One change at a time. Verify after each.
- File -> Find -> Replace format for all edits.

## Key Decisions (Locked)

1. Camera approach: Option B (custom CameraX). Multi-snap live sorting requires it. Option A (system camera) cannot produce the interaction. See CAMERA_SPEC.md.
2. Personal use only. Distribution (Play Store, testers, signed builds) deferred until explicitly requested.
3. Storage: app-private (filesDir/IndigiSnap/). Shared storage migration deferred.
4. File naming: immediate_folder + simplified_date + extension.
5. Recents: persistent (file-backed .indigisnap_recents.json).
6. Upload button placement: controls bar, next to Sort/Filter.
7. Logo: BASE_DIR/logo/ folder. Prefer logo.<ext>, else alphabetical first.

## Open Decisions

See DECISIONS_OPEN.md. Current outstanding items:
- File naming date format
- Share-target semantics (copy-only vs. delete-original)
- Multi-snap review screen architecture

## Working Rules For Assistants

- Do not suggest installing tools locally on a work computer.
- Do not use cat << EOF. Use python3 << 'PYEOF'.
- One step at a time. Verify before moving on.
- Do not skip steps. If something failed earlier and was patched around, flag it.
- Read TODO.md, DECISIONS_OPEN.md, ARCHITECTURE.md before starting new work.
- When the user says "go", execute the approved plan.
- Honest about scope. If it hasn't been tested on the phone, it doesn't work yet.
- Do not resolve open decisions silently. Ask.
- Do not defer cheap or useful features to v2 to save effort. Batch them into v1.

## Repository

github.com/calvenwilliams1-pixel/IndigiSnap

Package: com.indigisnap.app
Version: 0.1.0
minSdk: 24 (Android 7.0)
targetSdk: 34 (Android 14)
