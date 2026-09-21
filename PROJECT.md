# IndigiSnap — Project Context

**Read this first if you are an AI assistant picking up this project cold.**

---

## What This Is

IndigiSnap is a personal, single-user photo and video gallery app for Android. It started life as a Python/Flask web app that ran inside Termux on a phone, launched by Tasker, viewed in a browser. It was fragile — Termux would die, Tasker would misfire, Android would kill the server.

**The current goal:** convert it into a real, standalone Android APK. Hybrid architecture:

- **Android shell** (Kotlin): MainActivity + Foreground Service + WebView
- **Backend** (Go): compiled as a JNI shared library, runs inside the app's process
- **UI** (HTML/CSS/JS): the original Flask frontend, served by the Go backend, rendered in the WebView
- **Storage**: app-private directory for now (upgradeable to shared MediaStore later)
- **Server**: binds to 127.0.0.1 only — no network exposure

The end state: an app icon in the drawer. Tap it. The app launches, starts its own embedded Go server, and loads your photo gallery in a WebView. No Termux, no Tasker, no browser chrome.

---

## Development Constraints (Critical)

- **Dev environment is GitHub Codespaces.** Browser-based. No local installs. Ever.
- **Work computer is locked down** — no external uploads, no local tooling.
- **The user's phone holds all originals.** Files get to the Codespace via GitHub.
- **Terminal only.** The user prefers `python3 << 'PYEOF'` heredocs for writing files. `cat << EOF` heredocs corrupt on the Codespace web terminal and should never be used.
- **One step at a time.** Verify each step before proceeding. Do not bundle 5 commands into one block.
- **Terse responses preferred.** No long explanations unless asked.
- **The user is sharp** — catches inconsistencies, notices skipped steps, wants honesty about what's proven vs. assumed.

---

## Current State (as of this commit)

See `TODO.md` for the live status.
See `ARCHITECTURE.md` for the full file layout and purpose of each file.

At a glance:
- Go backend works in the browser (Codespaces port forwarding)
- Full browse handler with sort/filter/favorites
- View handler serves images and videos
- Action routes: create folder, delete, rename, toggle favorite, set sort
- Template renders the real IndigiSnap UI (neon/psychedelic theme)
- **Not yet embedded in the APK.** The Android shell still shows hardcoded HTML.

---

## The Original Flask Code

The original Python source (Flask routes, HTML interface, EXIF handling, etc.) was provided to the assistant at the start of this project. If you need to check the original behavior of any route, ask the user to paste the relevant section — it lives on their phone and has not been committed to the repo.

---

## Key Decisions (Locked In)

1. **Go, not Python, for the backend.** Cross-compiles to a static ARM64 binary. No runtime dependency. Survives Android Doze.
2. **JNI `c-shared`, not a spawned process.** Go code runs inside the app's process via `//export StartServer`.
3. **Foreground service with `dataSync` type** to keep the server alive when the app is backgrounded.
4. **App-private storage first** (`filesDir/IndigiSnap/`). Shared MediaStore later.
5. **`html/template` with `//go:embed`** for the UI. Jinja2 → Go template syntax conversion was done manually.
6. **Bind to 127.0.0.1 only.** Loopback. No LAN exposure.
7. **FFmpeg is optional.** Video helpers shell out to `ffprobe`/`ffmpeg` if present; otherwise no-op. Bundling a static ARM binary is a later phase.
8. **ARM64 + ARMv7 ABIs.** No x86_64 in release builds.

---

## Rules For The Assistant

- Do not suggest installing anything locally on the user's work computer.
- Do not use `cat << EOF`. Use `python3 << 'PYEOF'` for writing files.
- Verify after each step. Paste outputs. Confirm before moving on.
- Do not skip steps. If something failed earlier and was patched around, flag it.
- Read `TODO.md` and `ARCHITECTURE.md` before starting any new work.
- When the user says "go", execute the next item on the TODO list.
- Honest about what's proven vs assumed. If it hasn't been tested on the phone, it doesn't work yet.

---

## Repository

https://github.com/calvenwilliams1-pixel/IndigiSnap

**Branch:** main
**Package name:** com.indigisnap.app
**Current version:** 0.1.0
**minSdk:** 24 (Android 7.0)
**targetSdk:** 34 (Android 14)
