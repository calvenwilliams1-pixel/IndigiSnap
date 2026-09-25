# IndigiSnap - TODO

Live status. Update after every meaningful commit.

Legend: [x] done - [~] in progress - [ ] not started

## Phase 0 - Stabilize Current Build

- [x] APK builds and installs
- [x] Real UI renders on phone
- [x] Core browse/view/favorites/batch operations work
- [x] Camera and video buttons open system camera (one-shot)
- [ ] CameraX multi-snap flow (replaces one-shot)

## Batch 1 - Backend Features - COMPLETE

- [x] 1.1 Real EXIF reading (goexif)
- [x] 1.2 EXIF orientation (disintegration/imaging)
- [x] 1.3 Logo route
- [x] 1.4 Recent folders tracking

## Batch 2 - UI Features

- [x] 2.1 Logo in header
- [x] 2.2 Recents pills
- [x] 2.3 Folder browser modal
- [x] 2.4 Discreet upload button
- [ ] 2.4b Upload filter (images-only / videos-only)
- [ ] 2.5 Gallery viewer enhancements (zoom, buttons, counter, info panel)
- [ ] 2.6 Folder-level rotate
  - [ ] Design review: EXIF orientation interaction
  - [ ] Backend: POST /rotate_picture/<path>
  - [ ] UI: rotate button in viewer
- [ ] 2.7 Set logo UI
  - [ ] Backend: POST /set_logo (with MIME + size validation)
  - [ ] UI: Set Logo menu item in root folder

## Batch 3 - Kotlin Changes

- [ ] 3.1 Splash screen
- [ ] 3.2 Upload move semantics (move, not copy)
- [ ] 3.4 Share target integration (pending decision on semantics)
- [ ] 3.5 Splash reads logo
- [ ] 3.6 Permission prompt fix (ask once, not every launch)

## Batch 4 - CameraX Multi-Snap (in progress)

Spec: CAMERA_SPEC.md

- [ ] 4.1 CameraX foundation (preview + shutter + close)
- [ ] 4.2 Rapid-fire shooting (inbox writes)
- [ ] 4.3 Thumbnail strip + swipe review
- [ ] 4.4 Commit to folder (move from inbox, rename)
- [ ] 4.5 Crash recovery (orphaned inbox prompt)
- [ ] 4.6 Video capture + thumbnail generation on commit
- [ ] 4.7 Full camera controls (zoom, front/back, flash, resolution, night mode)
- [ ] 4.8 Orientation handling (session persists, per-shot EXIF)
- [ ] 4.9 WebView integration (JS bridge)
- [ ] 4.10 Test matrix on Samsung Android 14

## Batch 6 - Polish

- [ ] 6.1 Session memory + per-folder notes + debug overlay (L1+L2)
- [ ] 6.2 EXIF formatted strings + EXIF-aware batch rename
- [ ] 6.3 Search (folder-scoped + global)
- [ ] 6.4 Video thumbnail: delete-on-video-delete
- [ ] 6.5 Video thumbnail: orphan sweep on /browse
- [ ] 6.6 Video thumbnail: regen on demand if missing

## Section 4 - Distribution (Held By User)

Activated only when user says go.

- [ ] Signed release build + keystore
- [ ] Tester distribution
- [ ] Play Store requirements
- [ ] Shared storage migration
- [ ] Accessibility pass
- [ ] Tester documentation

## Deferred (Do Not Build) - User Must Approve

- pHash / near-duplicate clustering
- Rating/flag system
- Config export/import
- Side-by-side compare in viewer

## Rules

- No deferring work because it is hard. All pending features ship in v1.
- No resolving open decisions silently. Ask.
- Local Steam Deck build preferred (see RESUME.md for environment setup).

## Current Blocker / Next Action

**Next action:** Batch 4.1 - CameraX foundation.
- Add CameraX dependencies to app/build.gradle.kts
- Create CameraActivity.kt with preview surface
- Add CAMERA + RECORD_AUDIO permissions to manifest
- Test preview + single shot + return to WebView

**Prerequisite:** Steam Deck Android SDK install (in progress - resume at hotel wifi).
- sdkmanager "platform-tools" "platforms;android-34" "build-tools;34.0.0" "ndk;25.2.9519653"
- create local.properties
- ./gradlew assembleDebug

## Open Decisions

- Share-target semantics: copy-then-delete (Option 1) vs copy-only (Option 2)
See DECISIONS_OPEN.md for details.

## Quick Reference Commands

Run server locally on Steam Deck (port 8090):
  cd ~/IndigiSnap/go-backend
  INDIGISNAP_PORT=8090 INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server

Kill running server:
  source ~/.bashrc
  killindigi

Build APK locally (Steam Deck):
  cd ~/IndigiSnap
  ./gradlew assembleDebug
  adb install -r app/build/outputs/apk/debug/app-debug.apk

Build APK via CI:
  git push
  # wait for Actions, download artifact

Watch phone logs:
  adb logcat -c
  adb logcat -v threadtime | grep IndigiSnapDebug

Copy command output to clipboard:
  cb <command>

Repo location on Steam Deck: /home/deck/IndigiSnap
