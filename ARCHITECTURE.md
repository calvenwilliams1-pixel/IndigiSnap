# IndigiSnap - Architecture

Repository layout, file purposes, and runtime constraints.

## Repository Root

- PROJECT.md - context prompt for AI assistants
- TODO.md - live status tracker
- ARCHITECTURE.md - this file
- PRODUCTION_PLAN.md - v1 roadmap
- PRIORITIZATION_NOTES.md - batch plan and sequencing
- DECISIONS_OPEN.md - unresolved decisions and rationale
- CAMERA_SPEC.md - camera capture spec
- RESUME.md - cold-start orientation checklist
- DEBUG_LOG.md - session debug notes
- README.md - placeholder
- MASTER_PROMPT.md - older context file (superseded by PROJECT.md)
- build.gradle.kts - root Gradle build
- settings.gradle.kts - project name, modules
- gradle.properties - JVM args, AndroidX
- gradlew, gradlew.bat - wrapper scripts
- local.properties - sdk.dir, gitignored
- .gitignore - build artifact and local config exclusions

## Dev Environment

- .devcontainer/devcontainer.json - Codespaces container spec (Java 17, Go 1.22)
- .devcontainer/setup.sh - post-create hook, installs Android SDK and NDK
- .github/workflows/build.yml - CI (builds APK on push)

## Android Shell - app/

- app/build.gradle.kts - app module build config, jniLibs source set
- app/proguard-rules.pro - ProGuard rules (empty)
- app/src/main/AndroidManifest.xml - permissions, activity, service declarations
- app/src/main/java/com/indigisnap/app/MainActivity.kt
  - WebView host, WebChromeClient for file chooser, JNI loader
- app/src/main/java/com/indigisnap/app/ServerBridge.kt
  - JNI bridge: loads libindigisnap.so, external fun StartServer/StopServer
- app/src/main/java/com/indigisnap/app/ServerService.kt
  - Foreground service that starts the Go server on app launch
- app/src/main/jniLibs/arm64-v8a/libindigisnap.so - compiled Go (64-bit)
- app/src/main/jniLibs/armeabi-v7a/libindigisnap.so - compiled Go (32-bit)
- app/src/main/res/ - Android resources (icons later)

## Go Backend - go-backend/

- go-backend/go.mod - module definition
- go-backend/go.sum - dependency checksums
- go-backend/IndigiSnap/ - test media folder (gitignored)
- go-backend/cmd/server/main.go - HTTP server entry, buildMux()
- go-backend/cmd/server/jni.go - JNI exports for Kotlin
- go-backend/internal/security/security.go
  - IsSafePath, ValidatePath, SanitizeFilename, AllowedFile, IsVideoFile, IsHiddenFolder
- go-backend/internal/meta/meta.go
  - Folder metadata, favorites, recents, breadcrumbs
- go-backend/internal/ui/ui.go
  - TemplateData struct, Render() via html/template
- go-backend/internal/ui/interface.html
  - Full app UI (HTML/CSS/JS), embedded via go:embed
- go-backend/internal/video/video.go
  - ffprobe and ffmpeg wrappers (no-op if binaries missing)
- go-backend/internal/handlers/browse.go - /browse, favorites virtual folder, previews
- go-backend/internal/handlers/view.go - /view/<path>
- go-backend/internal/handlers/actions.go - create/rename/delete folders, upload, sort, favorite
- go-backend/internal/handlers/batch.go - batch_delete, batch_move, batch_rename, export_zip
- go-backend/internal/handlers/browser.go - /folder_browser
- go-backend/internal/handlers/dupes.go - /find_duplicates
- go-backend/internal/handlers/info.go - /image_info (STUB until Batch 1.1)

## Registered HTTP Routes

| Method | Route | Handler | Status |
|--------|-------|---------|--------|
| GET | /health | inline | done |
| GET | / | inline | done |
| GET | /browse | BrowseHandler | done |
| GET | /browse/<path> | BrowseHandler | done |
| GET | /browse/favorites | BrowseHandler (virtual) | done |
| GET | /view/<path> | ViewHandler | done |
| GET | /image_info/<path> | InfoHandler | STUB |
| GET | /folder_browser | BrowserHandler | done |
| GET | /find_duplicates/<path> | DupesHandler | done |
| GET | /logo/<path> | - | TODO (Batch 1.3) |
| GET | /search | - | TODO (Batch 6.3) |
| POST | /upload | ActionHandler.Upload | done (copies; move is Batch 3.2) |
| POST | /create_folder/<path> | ActionHandler | done |
| POST | /rename_folder/<path> | ActionHandler | done |
| POST | /delete_folder/<path> | ActionHandler | done |
| POST | /delete_picture/<path> | ActionHandler | done |
| POST | /toggle_favorite/<path> | ActionHandler | done |
| POST | /set_sort/<path> | ActionHandler | done |
| POST | /batch_delete | BatchHandler | done |
| POST | /batch_move | BatchHandler | done |
| POST | /batch_rename | BatchHandler | done |
| POST | /export_zip | BatchHandler | done |
| POST | /export_folder_zip/<path> | BatchHandler | done |
| POST | /rotate_picture/<path> | - | TODO (Batch 2.6) |
| POST | /set_logo | - | TODO (Batch 2.7) |
| POST | /set_notes/<path> | - | TODO (Batch 6.1) |

## JNI Interface

Go exports:
- Java_com_indigisnap_app_ServerBridge_StartServer(env, clazz, port, baseDir)
- Java_com_indigisnap_app_ServerBridge_StopServer(env, clazz)

Kotlin declares:
- external fun StartServer(port: Int, baseDir: String)
- external fun StopServer()

Cross-compile (in CI):
  NDK=$ANDROID_HOME/ndk/25.2.9519653
  TOOLCHAIN=$NDK/toolchains/llvm/prebuilt/linux-x86_64
  CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC=$TOOLCHAIN/bin/aarch64-linux-android24-clang go build -buildmode=c-shared -o libindigisnap.so ./cmd/server

## Data On Disk

Per-folder metadata: .indigisnap_meta.json
  Fields: thumb, active_prefix, history, sort, theme, previews, created_at, updated_at, hide_from_app, notes (added in Batch 6.1)

Global favorites: .indigisnap_favorites.json
  JSON array of relative paths

Global recents: .indigisnap_recents.json (added in Batch 1.4)
  JSON array of { name, path }

Base directory (dev): /home/deck/IndigiSnap/go-backend/IndigiSnap/
Base directory (phone): /data/data/com.indigisnap.app/files/IndigiSnap/

## Environment Variables

- INDIGISNAP_PORT default 8080
- INDIGISNAP_BASE_DIR default ./IndigiSnap

## Runtime Constraints

Non-obvious facts that trip up future changes.

### Single WebView
The app hosts exactly ONE WebView (in MainActivity.kt), loading
http://127.0.0.1:8080/browse and never navigating away. All "screens"
(modals, viewers, panels) are JS view toggles over the same DOM.

Consequences:
- JS state survives view transitions; nothing is freed by closing a panel
- Temporary collections (pendingSnaps, selection arrays) need explicit
  reset on discard/cancel - see DECISIONS_OPEN.md multi-snap note
- Every "screen" is CSS display toggling, not a page load

### Back button behavior
MainActivity overrides onBackPressed: WebView back if history exists,
otherwise exit. NOTE: JS-only view changes (modals, viewers) do not
create history entries, so back does NOT close them. Modals must
provide their own close button. Modal-dismiss-via-back would need
JS bridge work - not yet implemented.

### Loopback-only server
Go server binds 127.0.0.1 only, not exposed to the network.

### App-private storage only
Everything under filesDir/IndigiSnap - invisible to camera roll and
other apps. No runtime storage permissions needed for the app's own
folder. Shared storage migration is deferred (see DECISIONS_OPEN.md).

### JNI boundary is hard
Go is called through libindigisnap.so with two entry points only
(StartServer, StopServer). New Kotlin to Go calls need new exports on
both sides - do not assume more surface exists.

### Cross-compilation is CI-only
The Android .so is built by GitHub Actions. Local go run uses host
architecture - behavior can differ (paths, syscalls). Test in CI when
unsure.

### Inbox folder (pending, Batch 4)
Multi-snap camera captures write to filesDir/IndigiSnap/.inbox/<session-id>/
on capture, written by Kotlin directly. JS holds only filenames. Discard
deletes the folder; commit moves files to destination folder. See
CAMERA_SPEC.md items 1 and 5.

## Build And Run

### Local in Codespaces or Steam Deck
  cd go-backend
  INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server
  Open http://127.0.0.1:8080/browse

### APK build (CI)
  Triggered by push to main, or manual dispatch
  Artifact: indigisnap-debug

### Install on phone
  adb install -r app-debug.apk

### Logcat debug
  adb logcat -c
  adb logcat -v threadtime | grep IndigiSnapDebug

## Key Constraints

- Server binds 127.0.0.1 only
- All paths validated against BaseDir
- Hidden folders excluded from UI and protected from deletion
- App-private storage
- No ffmpeg dependency - video helpers no-op gracefully
- ARM64 + ARMv7 only (no x86_64 in release)
