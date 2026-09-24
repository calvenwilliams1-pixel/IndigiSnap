# IndigiSnap - TODO

Live status. Update after every meaningful commit.

Legend: [x] done - [~] in progress - [ ] not started

## Phase 0 - Stabilize Current Build

- [x] APK builds and installs
- [x] Real UI renders on phone
- [x] Core browse/view/favorites/batch operations work
- [~] Camera buttons exist but do not open a real camera session
- [ ] Known-good tag v0.2.0-stable

## Batch 1 - Backend Features (offline-testable in browser)

- [ ] 1.1 Real EXIF reading (goexif)
- [ ] 1.2 EXIF orientation (disintegration/imaging)
- [ ] 1.3 Logo route
- [ ] 1.4 Recent folders tracking

## Batch 2 - UI Features (offline-testable in browser)

- [ ] 2.1 Logo in header
- [ ] 2.2 Recents pills
- [ ] 2.3 Folder browser modal
- [ ] 2.4 Discreet upload button
- [ ] 2.5 Gallery viewer enhancements (zoom, buttons, counter, info)
- [ ] 2.6 Folder-level rotate
- [ ] 2.7 Set logo UI

## Batch 3 - Kotlin Changes (need build to test)

- [ ] 3.1 Splash screen
- [ ] 3.2 Upload move semantics
- [ ] 3.3 Camera capture wiring (CameraX)
- [ ] 3.4 Share target integration
- [ ] 3.5 Splash reads logo (deferred)

## Batch 4 - Camera Implementation (CameraX)

Spec: CAMERA_SPEC.md

- [ ] CameraX dependencies + preview surface
- [ ] Shutter + rapid-fire writes to inbox (item 1)
- [ ] Session persistence across orientation (item 2)
- [ ] Review grid with select-to-delete (item 3)
- [ ] Commit = move + delete inbox (item 4)
- [ ] Crash recovery prompt (item 5)
- [ ] Feature parity: zoom, camera switch, flash (item 6)
- [ ] Night mode with hardware-aware disabled state (item 6)
- [ ] Test matrix on Samsung Android 14

Do not start until:
- [ ] File naming format decided (DECISIONS_OPEN.md)
- [ ] Multi-snap review architecture decided (DECISIONS_OPEN.md)
- [x] CAMERA_SPEC.md committed

## Batch 6 - Trivial Wins + Polish

- [ ] 6.1 Session memory + per-folder notes + debug overlay (L1+L2)
- [ ] 6.2 EXIF formatted strings + EXIF-aware batch rename
- [ ] 6.3 Search (folder-scoped + global)

## Section 4 - Only If Distributing (Dormant)

Do not build unless user explicitly activates distribution.

- [ ] Signed release build + keystore
- [ ] Tester distribution (Firebase or Play Internal Testing)
- [ ] Play Store requirements
- [ ] Shared storage migration
- [ ] Accessibility pass
- [ ] Tester documentation

## Deferred (Do Not Build in v1)

- Shared storage migration (see above)
- EXIF write support
- pHash / near-duplicate clustering
- Rating/flag system
- Config export/import
- Splash reads logo (3.5)
- Side-by-side compare in viewer

## Current Blocker / Next Action

**Next action:** Batch 1.1 - Real EXIF reading in /image_info.
- Update go-backend/internal/handlers/info.go
- Add github.com/rwcarlsen/goexif/exif dependency
- Test with curl in Steam Deck browser
- Commit locally

## Open Decisions (Blocking Specific Batches)

- File naming date format - blocks Batch 4
- Share-target semantics - blocks Batch 3.4
- Multi-snap review architecture - blocks Batch 4

See DECISIONS_OPEN.md for full list.

## Quick Reference Commands

Run server locally (Steam Deck or Codespaces):
  cd go-backend
  INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server

Build APK (CI):
  git push
  # wait for Actions, download artifact

Install APK (via ADB):
  adb install -r app-debug.apk

Watch logs:
  adb logcat -c
  adb logcat -v threadtime | grep IndigiSnapDebug

Verify Go compiles:
  cd go-backend && go vet ./... && go build ./cmd/server

Repo location on Steam Deck: /home/deck/IndigiSnap
