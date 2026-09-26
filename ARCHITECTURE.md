# IndigiSnap — Architecture

Repository layout, file purposes, runtime constraints.

---

## Repository Root

- MASTER_PROMPT.md — context for AI assistants
- TODO.md — live status and queued work
- ARCHITECTURE.md — this file
- PROJECT.md — original context (superseded by MASTER_PROMPT.md)
- CAMERA_SPEC.md — camera behavior spec
- DECISIONS_OPEN.md — resolved and open decisions
- PRIORITIZATION_NOTES.md — batch plan
- PRODUCTION_PLAN.md — v1 roadmap
- RESUME.md — cold-start orientation
- CHEATSHEET.md — quick commands
- README.md, DEBUG_LOG.md — historical
- build.gradle.kts, settings.gradle.kts, gradle.properties
- gradlew, gradlew.bat, local.properties (gitignored)
- .gitignore, .devcontainer/, .github/workflows/

---

## Android Shell — app/

### Kotlin files in app/src/main/java/com/indigisnap/app/

**MainActivity.kt**
- WebView host, splash, permission gate
- JS bridge (WebAppBridge inner class): openCamera(folder),
  shareFile(relPath, name), shareFiles(jsonArray)
- registerForActivityResult for camera launcher and permission
  launcher
- Legacy camera path via launchCameraIntent for LEGACY_SINGLE and
  LEGACY_FALLBACK
- shouldOverrideUrlLoading with parameterized allowedHosts
- logOrphanedInboxes on startup
- TODO: SAF folder picker integration

**CameraActivity.kt**
- Full-screen CameraX preview
- Top bar: ❌ close, counter, ⚡ flash, 🔄 switch, ✅ done
- Bottom: shot counter, thumbnail strip, shutter button
- Review overlay: ❌ close, ✅ commit, ↺ 🗑️ ↻, counter
- Multi-select mode: long-press entry, checkbox overlays, batch
  toolbar
- Pinch + pan + double-tap + swipe gestures
- attemptClose, commitSession, discardSession
- onBackPressedCallback routes through attemptClose or
  closeReviewOverlay

**CameraController.kt**
- All CameraX wiring (bind, capture, rotation, warm-up, timeouts)
- Shot session state (List<SessionShot>)
- rotateShot (needs verification pass), deleteShots (to be repurposed),
  toggleShotDeletion, toggleBatchSelection, clearBatchSelection
- loadOrientedBitmap (to be split into loadThumbnail and
  loadFullResolution)
- focusAt (queued), setZoomRatio, updateTargetRotation
- Diagnostic logging (to be removed after Pass Q)

**SessionState.kt**
- Parcelable payload between MainActivity and CameraActivity
- Fields: sessionId, folder, shotCount, lastShotPath, sessionFinished,
  fallbackUsed, failureReason, cameraStartTimeMs

**SessionShot.kt**
- Data class for a shot in the current session
- Fields: file, capturedAt, markedForDeletion, thumbnail,
  selectedForBatch

**ServerService.kt**
- Foreground service, starts on app launch
- Currently hardcodes filesDir/IndigiSnap as base dir
- To be changed to read from Intent extra

**ServerBridge.kt**
- JNI loader for libindigisnap.so
- StartServer(port, baseDir), StopServer()

### Resources

- AndroidManifest.xml — permissions, activities, service, provider
- res/xml/file_paths.xml — FileProvider paths. Currently covers
  camera_cache and .inbox. Needs media root entry.
- res/drawable/ — currently empty. Needs 8 vector icons (queued).
- res/mipmap-*/ — default Android icons (custom art deferred)

### Native libraries

- jniLibs/arm64-v8a/libindigisnap.so — built by CI or locally via
  cross-compile with NDK 25.2.9519653

---

## Go Backend — go-backend/

- cmd/server/main.go — entry point, buildMux, routes
- cmd/server/jni.go — JNI exports: Java_com_indigisnap_app_ServerBridge_StartServer,
  StopServer
- internal/security/security.go — path safety, sanitization, file type
  checks
- internal/meta/meta.go — FolderMeta, favorites, recents, breadcrumbs,
  GetLogoPath/GetLogoURL
- internal/ui/ui.go — TemplateData, Render
- internal/ui/interface.html — the WebView UI (HTML/CSS/JS)
- internal/video/video.go — ffprobe/ffmpeg wrappers, CleanOrphanThumbnails
- internal/handlers/browse.go — /browse (with folder previews,
  CleanOrphanThumbnails call)
- internal/handlers/view.go — /view, EXIF orientation, JPEG quality
  100 (to be 90)
- internal/handlers/actions.go — create/rename/delete/upload/sort/
  favorite/rotate
- internal/handlers/batch.go — batch_delete/batch_move/batch_rename/
  export_zip
- internal/handlers/browser.go — /folder_browser
- internal/handlers/dupes.go — /find_duplicates
- internal/handlers/info.go — /image_info (real EXIF via goexif)
- internal/handlers/logo.go — /logo, /set_logo

---

## Registered HTTP Routes

| Method | Route | Handler | Status |
|--------|-------|---------|--------|
| GET | /health | inline | done |
| GET | / | inline | done |
| GET | /browse | BrowseHandler | done |
| GET | /browse/<path> | BrowseHandler | done |
| GET | /browse/favorites | BrowseHandler (virtual) | done |
| GET | /view/<path> | ViewHandler | done |
| GET | /image_info/<path> | InfoHandler | done |
| GET | /folder_browser | BrowserHandler | done |
| GET | /find_duplicates/<path> | DupesHandler | done |
| GET | /logo/<path> | LogoHandler | done |
| POST | /set_logo | LogoHandler | done |
| POST | /upload | ActionHandler | done |
| POST | /create_folder/<path> | ActionHandler | done |
| POST | /rename_folder/<path> | ActionHandler | done |
| POST | /delete_folder/<path> | ActionHandler | done |
| POST | /delete_picture/<path> | ActionHandler | done |
| POST | /toggle_favorite/<path> | ActionHandler | done |
| POST | /set_sort/<path> | ActionHandler | done |
| POST | /rotate_picture/<path> | ActionHandler | done |
| POST | /batch_delete | BatchHandler | done |
| POST | /batch_move | BatchHandler | done |
| POST | /batch_rename | BatchHandler | done |
| POST | /export_zip | BatchHandler | done |
| POST | /export_folder_zip/<path> | BatchHandler | done |

---

## Data On Disk (Current — App Private)

Base dir on phone: /data/data/com.indigisnap.app/files/IndigiSnap/

- .indigisnap_meta.json — per folder metadata
- .indigisnap_favorites.json — global favorites
- .indigisnap_recents.json — recent folders
- .inbox/<session-id>/IMG_<epoch_ms>_<seq>.jpg — pending camera shots
- logo/logo.<ext> — custom logo
- {folder}/{files} — committed media
- {folder}/.thumbs/<video_basename>.jpg — video thumbnails

Planned (after SAF folder picker):
- /sdcard/Pictures/IndigiSnap/ (user-chosen or default)

---

## Runtime Constraints

### Single WebView
The app hosts one WebView in MainActivity. All browse-page "screens"
(viewer, modals, panels) are JS view toggles over the same DOM.

### Back button
MainActivity onBackPressed routes through attemptClose (camera) or
closeReviewOverlay. CameraActivity uses OnBackPressedCallback.

### Loopback server
Go server binds 127.0.0.1 only.

### App-private storage
Currently filesDir/IndigiSnap. SAF folder picker to move to
/sdcard/Pictures/IndigiSnap.

### JNI boundary
Go called via libindigisnap.so. Two exports: StartServer, StopServer.
Cross-compiled by GitHub Actions or locally with NDK.

### Camera flow
- Photos: CameraActivity + CameraController, multi-shot session
- Video: MediaStore.ACTION_VIDEO_CAPTURE via MainActivity
- Fallback: if CameraX fails, MainActivity uses legacy system camera

### Threading rules
- All UI state mutations post to main handler
- Bitmap decode on camera executor
- Rotation post-processing on camera executor
- Every disk write verified before use

---

## Build And Run

### Local Go server (Steam Deck, port 8090):
cd ~/IndigiSnap/go-backend
INDIGISNAP_PORT=8090 INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server

### Local APK build:
cd ~/IndigiSnap
./gradlew assembleDebug

### Install:
adb install -r app/build/outputs/apk/debug/app-debug.apk

### Watch logs:
adb logcat -c
adb logcat -v threadtime | grep -iE "IndigiSnap|chromium"

### Cross-compile Go for Android ARM64:
cd ~/IndigiSnap
NDK=$ANDROID_HOME/ndk/25.2.9519653
TOOLCHAIN=$NDK/toolchains/llvm/prebuilt/linux-x86_64
cd go-backend
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
  CC=$TOOLCHAIN/bin/aarch64-linux-android24-clang \
  go build -buildmode=c-shared \
  -o ../app/src/main/jniLibs/arm64-v8a/libindigisnap.so ./cmd/server

---

## Key Constraints

- Server binds 127.0.0.1 only
- All paths validated against BaseDir
- Hidden folders (dot prefix, logo) excluded
- App-private storage for now
- ARM64 + ARMv7 only
- No continuous animations (battery/GPU)
- Every disk write verified
- Sampled decode for thumbnails
- One build per pass, no mixed correctness + styling
