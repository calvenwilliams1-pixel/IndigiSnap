# IndigiSnap — TODO

**Live status. Update after every meaningful commit.**

Legend: [x] done · [~] in progress · [ ] not started

---

## Phase 0 — Repo Housekeeping
- [x] Create GitHub repo (calvenwilliams1-pixel/IndigiSnap)
- [x] Add .gitignore (Gradle, build artifacts, local.properties, test media)
- [x] Add MASTER_PROMPT.md (superseded by PROJECT.md)

## Phase 1 — Go Backend Skeleton
- [x] Devcontainer with Android SDK, NDK, Java 17, Go 1.22
- [x] Gradle wrapper, Android project structure
- [x] Minimal APK builds, installs, shows hardcoded neon splash
- [x] go-backend module initialized
- [x] cmd/server/main.go with /health route
- [x] Cross-compile to Android ARM64 verified (7 MB static binary)

## Phase 2 — Port Routes From Flask to Go
- [x] internal/security/ — path safety, sanitization, file type checks
- [x] internal/meta/ — folder metadata, favorites, breadcrumbs, prefix generation
- [x] internal/ui/ — html/template + //go:embed, real UI in interface.html
- [x] internal/video/ — ffprobe/ffmpeg wrappers, graceful no-op if missing
- [x] handlers/browse.go — folder + media listing, sort, filter, favorites
- [x] handlers/view.go — serves image/video bytes with cache headers
- [x] handlers/actions.go — create folder, rename, delete, favorite toggle, set sort
- [x] Register all routes in main.go
- [x] Verify in Codespaces browser: real UI renders, tiles work, sort/filter work

## Phase 3 — Embed Go Into The APK
- [ ] Add //export StartServer and //export StopServer to Go
- [ ] Compile Go as c-shared for arm64-v8a and armeabi-v7a
- [ ] Drop .so files into app/src/main/jniLibs/
- [ ] Create ServerBridge.kt with System.loadLibrary("indigisnap")
- [ ] Create ServerService.kt (foreground service)
- [ ] Update MainActivity.kt to start service and load http://127.0.0.1:8080
- [ ] Update AndroidManifest.xml with service + permissions
- [ ] Build APK and install on phone
- [ ] Verify: real UI on phone, no browser chrome, all features work

## Phase 4 — Polish
- [ ] Custom app icon (512x512 source → all mipmap densities)
- [ ] Add android:icon to manifest
- [ ] Splash screen via androidx.core:core-splashscreen
- [ ] theme-color meta for status bar

## Phase 5 — GitHub Actions CI
- [ ] .github/workflows/build.yml — build APK on tag push
- [ ] Upload APK as artifact
- [ ] Optional: signed release build via secrets

## Phase 6 — Real Storage (later)
- [ ] Shared MediaStore access to /sdcard/Pictures/IndigiSnap/
- [ ] READ_MEDIA_IMAGES / READ_MEDIA_VIDEO permissions
- [ ] Android Photo Picker for uploads
- [ ] One-time consent flow

## Phase 7 — Unported Routes (later)
- [ ] POST /upload
- [ ] POST /batch_delete
- [ ] POST /batch_move
- [ ] POST /batch_rename
- [ ] POST /export_zip
- [ ] POST /export_folder_zip
- [ ] GET /find_duplicates
- [ ] GET /image_info (EXIF reader)
- [ ] GET /folder_browser
- [ ] GET /logo

## Phase 8 — Play Store Prep (optional)
- [ ] Privacy policy
- [ ] Store assets (icon, feature graphic, screenshots)
- [ ] AAB build
- [ ] Data Safety form
- [ ] Photo/Video Permissions Declaration
- [ ] Internal test → closed test → production

---

## Current Blocker / Next Action

**Next action:** Start Phase 3. Write the JNI bridge in Go, compile as c-shared, wire up the Kotlin shell, build the APK.

**Deadline pressure:** User leaves for Japan in ~2 days. Priority is a working APK on the phone before departure.

---

## Quick Reference Commands

Run the server locally (Codespaces):
    cd /workspaces/IndigiSnap/go-backend && INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server

Build the APK:
    cd /workspaces/IndigiSnap && ./gradlew assembleDebug
    # APK at app/build/outputs/apk/debug/app-debug.apk

Check everything compiles:
    cd /workspaces/IndigiSnap/go-backend && go vet ./...

Test media folder:
    /workspaces/IndigiSnap/go-backend/IndigiSnap/  (gitignored, contains test files)
