# IndigiSnap — Master Context Prompt

Paste this entire file at the start of a new chat to bring a fresh assistant up to speed on this project.

---

## Project Overview

IndigiSnap is a personal, single-user photo and video gallery app. It was originally built as a Flask web app (Python) that ran inside Termux on Android, launched by Tasker, and viewed in a mobile browser. The goal now is to convert it into a real, standalone Android APK that can eventually be published to the Play Store.

The original Flask source is preserved as a single `.txt` file (the user has it locally) and contains:
- Flask routes for browse, upload, delete, batch operations, favorites, ZIP export, folder browser, image info, duplicate finder
- An embedded HTML/CSS/JS UI (the full frontend)
- JSON-based metadata (.indigisnap_meta.json, .indigisnap_favorites.json)
- EXIF reading, EXIF rotation, ffmpeg video thumbnails
- Basic auth, path traversal protection, sanitization

---

## Architecture Decision (Locked In)

**Hybrid Android app:**
- Kotlin/Android shell (MainActivity + Foreground Service)
- Go backend compiled as a JNI shared library (`libindigisnap.so`)
- Existing HTML/CSS/JS UI served by the Go server, loaded into a WebView
- Server binds to `127.0.0.1` only (loopback, not exposed to network)
- Storage: app-private directory (`filesDir/IndigiSnap`) for now; upgradeable to MediaStore later
- No Termux, no Tasker, no browser chrome

**Why hybrid:** Reuses 100% of the existing UI (the hardest part), satisfies Play Store requirements, feels native enough, and the Go backend compiles to a tiny static binary that survives Android's Doze and process-killer behaviors.

---

## Development Environment

- **Dev:** GitHub Codespaces (browser-based, work computer, no local installs)
- **Repo:** https://github.com/calvenwilliams1-pixel/IndigiSnap
- **CI/CD:** GitHub Actions builds APKs on tag push
- **Toolchain in Codespace:** Java 17, Go 1.22, Android SDK 34, NDK 25.2.9519653, Gradle 8.2 (via wrapper)
- **Devcontainer:** `.devcontainer/devcontainer.json` + `.devcontainer/setup.sh` auto-install Android SDK on container rebuild

**Critical habits:**
- Always `cd /workspaces/IndigiSnap` (capital I)
- One step at a time, verify after each step
- Don't skip the "Rebuild Container" prompt after changing devcontainer files
- No long explanations unless asked — user prefers terse execution

---

## Current State (End of Week 1)

### What Works
- Real Android APK builds successfully via `./gradlew assembleDebug`
- APK is signed (v2 scheme), valid, and installs on the phone
- App appears in the drawer labeled "IndigiSnap" (default Android icon)
- Tapping it launches a WebView that shows a hardcoded neon "INDIGISNAP" splash with "Hello from IndigiSnap"
- Repo is committed and pushed to GitHub

### What Doesn't Work Yet
- No Go backend
- No real HTTP server
- No file access
- No custom icon
- No custom splash screen
- No photo permissions
- The hardcoded HTML in `MainActivity.kt` is a placeholder, not real UI

### Repo File Structure (verified)
```
/workspaces/IndigiSnap/
├── .devcontainer/
│   ├── devcontainer.json
│   └── setup.sh
├── .github/
│   └── workflows/          (empty — not yet created)
├── app/
│   ├── build.gradle.kts
│   ├── proguard-rules.pro
│   └── src/main/
│       ├── AndroidManifest.xml
│       ├── java/com/indigisnap/app/MainActivity.kt
│       └── res/            (mostly empty)
├── gradle/wrapper/
│   ├── gradle-wrapper.jar
│   └── gradle-wrapper.properties
├── .gitignore
├── build.gradle.kts        (root)
├── settings.gradle.kts
├── gradle.properties
├── gradlew
├── gradlew.bat
├── local.properties        (gitignored)
├── README.md
└── MASTER_PROMPT.md        (this file)
```

### Key Config Values (already set)
- Package name: `com.indigisnap.app`
- App label: `IndigiSnap`
- minSdk: 24 (Android 7.0)
- targetSdk: 34 (Android 14)
- compileSdk: 34
- Version: 0.1.0 / versionCode 1
- Permissions currently declared: only `INTERNET`
- MainActivity overrides `onBackPressed` to navigate WebView history

---

## Full Implementation Plan (Week 2+)

### Phase 0: Housekeeping (5 min)
- [ ] Ensure `.gitignore` covers `.gradle/`, `build/`, `local.properties`, `*.apk`, `app/build/`
- [ ] Commit and push

### Phase 1: Go Backend Skeleton (30 min)
- [ ] `mkdir -p go-backend/cmd/server go-backend/internal/{handlers,meta,security,ui,video}`
- [ ] `cd go-backend && go mod init github.com/calvenwilliams1-pixel/indigisnap`
- [ ] Write `cmd/server/main.go` — minimal HTTP server, `/health` route, binds to `127.0.0.1:8080`, base dir from env
- [ ] Test with `go run ./cmd/server` in Codespaces, forward port
- [ ] Verify `GOOS=android GOARCH=arm64 go build` produces a binary

### Phase 2: Port Routes From Flask to Go (2–3 hrs)
- [ ] `internal/security/` — port IsSafePath, ValidatePath, SanitizeFilename, AllowedFile, IsVideoFile, IsHiddenFolder
- [ ] `internal/meta/` — port GetMeta, SaveMeta, LoadFavorites, SaveFavorites, ToggleFavorite, CleanupFavorites, GenerateSearchablePrefix, GetBreadcrumbs
- [ ] `internal/ui/` — extract HTML_INTERFACE string to `interface.html`, use `//go:embed`, convert Jinja2 → Go templates
- [ ] `internal/video/` — GetDuration (ffprobe), GenerateThumbnail (ffmpeg), graceful no-op if binaries missing
- [ ] `internal/handlers/` — port every Flask route to a Go handler: Browse, View, Upload, CreateFolder, RenameFolder, DeleteFolder, DeletePicture, BatchDelete, BatchMove, BatchRename, ExportZip, ExportFolderZip, ToggleFavorite, ImageInfo, FindDuplicates, FolderBrowser, SetSort, ServeLogo
- [ ] Auth middleware (basic auth, env-gated)
- [ ] Wire everything in main.go
- [ ] Test every feature in Codespaces with sample images

### Phase 3: Embed Go Into The APK (1–2 hrs)
- [ ] Add `//export StartServer` and `//export StopServer` to Go, set `main()` empty, compile with `-buildmode=c-shared`
- [ ] Cross-compile for `arm64-v8a` and `armeabi-v7a` using NDK 25.2.9519653
- [ ] Drop `.so` files into `app/src/main/jniLibs/{arm64-v8a,armeabi-v7a}/`
- [ ] Kotlin `ServerBridge.kt` with `System.loadLibrary("indigisnap")` and external fun declarations
- [ ] `ServerService.kt` — foreground service, START_STICKY, posts a persistent notification, calls `ServerBridge.StartServer(8080, filesDir)`
- [ ] `MainActivity.kt` — start service, poll `/health`, then `webView.loadUrl("http://127.0.0.1:8080")`
- [ ] `AndroidManifest.xml` — add service, FOREGROUND_SERVICE, FOREGROUND_SERVICE_DATA_SYNC, POST_NOTIFICATIONS permissions
- [ ] Test on phone — real UI appears

### Phase 4: Real File Access (30 min)
- [ ] Point Go at `filesDir/IndigiSnap`
- [ ] Push sample images (via UI upload once it works, or adb push)
- [ ] Verify upload, browse, favorite, delete, batch ops, folder browser, export ZIP, image info all work

### Phase 5: Polish (30 min)
- [ ] Custom app icon (512×512 source → all mipmap densities)
- [ ] Add `android:icon="@mipmap/ic_launcher"` to manifest
- [ ] Splash screen via androidx.core:core-splashscreen
- [ ] `<meta name="theme-color" content="#07040d">` for status bar

### Phase 6: GitHub Actions (15 min)
- [ ] `.github/workflows/build.yml` — trigger on tag push, setup Java/Go/Android SDK/NDK, build `.so`, run `./gradlew assembleDebug` and `assembleRelease`, upload artifacts
- [ ] Optional: signed release via GitHub Secrets (keystore base64)
- [ ] Tag `v0.2.0` and verify APK appears in Actions artifacts

### Phase 7: Play Store Prep (optional, 1 hr)
- [ ] Add READ_MEDIA_IMAGES / READ_MEDIA_VIDEO / MediaStore access
- [ ] Android Photo Picker for uploads (no permission needed)
- [ ] Privacy policy (GitHub Pages)
- [ ] Store assets (icon, feature graphic, screenshots)
- [ ] `./gradlew bundleRelease` → AAB
- [ ] Data Safety form, Photo/Video Permissions Declaration
- [ ] Internal test → closed test → production

---

## Key Technical Decisions

1. **Go as JNI `c-shared`, not executable.** Runs inside the app's process, no separate binary to spawn. Exposes `StartServer(port, baseDir)` and `StopServer()`.
2. **Foreground service with `dataSync` type** to survive Doze.
3. **Bind to `127.0.0.1` only** — no network exposure, no firewall prompts.
4. **App-private storage first**, MediaStore later. Zero permission complexity in Week 2.
5. **HTML interface unchanged** — Flask Jinja2 → Go html/template is a mechanical translation. Same CSS, same JS, same behavior.
6. **Skip ffmpeg for now** — video thumbnails show 🎬 placeholder. Bundle a static ffmpeg ARM binary in a later phase if needed.
7. **ARM64 + ARMv7 only** — no x86_64 in release (only for emulator dev if ever needed).
8. **Play Store photo permissions policy** — must use Android Photo Picker or justify broad access. Plan for this from Phase 7.

---

## Notes For The Assistant

- User is on a **work computer**, uses **only GitHub + Codespaces**. No local installs. Do not suggest installing anything locally.
- User prefers **terse, execution-focused** responses. Long explanations are unwelcome unless asked. Give commands and verification steps.
- **One step at a time.** After each command, wait for output before proceeding. Do not bundle 5 commands into one block without checkpoints.
- **Verify before moving on.** Don't assume a step worked.
- **Codespace is ephemeral.** Push to GitHub frequently.
- User's terminal prompt shows `@calvenwilliams1-pixel ➜ /workspaces/IndigiSnap (main) $`
- User is enthusiastic and sharp — catches inconsistencies. Be honest about what's proven vs assumed.
- The original Flask source lives in a `.txt` file the user has locally. When porting, ask the user to paste specific sections if not already in context.

---

## Immediate Next Step

**Phase 0** — housekeeping commit, then **Phase 1, Task 1.1** — create the `go-backend/` directory structure and initialize the Go module.

Say "go" to begin Phase 0.