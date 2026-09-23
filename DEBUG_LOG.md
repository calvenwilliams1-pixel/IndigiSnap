# IndigiSnap Debug Log

## Current State (2026-09-22)

- APK builds successfully via GitHub Actions
- Both ARM64 and ARMv7 .so files ARE packaged in the APK (verified)
- App installs, launches, but shows a **white screen**
- No custom IndigiSnapDebug logs appear in logcat before the next build
- Adding debug logging to MainActivity, ServerService, ServerBridge

## What Works

- Go server runs fine in Codespaces
- All routes tested and working in browser
- Full IndigiSnap UI renders in browser
- APK build pipeline: green
- .so files correctly embedded in APK

## What's Broken

- White screen on phone
- Unknown root cause (pending next log capture)

## Debug Commands

Run server locally (Codespaces):
    cd /workspaces/IndigiSnap/go-backend && INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server

Capture phone logs (Windows PowerShell):
    cd C:\Users\calve\Downloads\platform-tools-latest-windows\platform-tools
    .\adb.exe logcat -c
    .\adb.exe logcat -v threadtime | Select-String -Pattern "IndigiSnapDebug"

Filter to app only:
    .\adb.exe logcat -d -v threadtime | Select-String "indigisnap"

## Key Facts

- Package: com.indigisnap.app
- Server port: 8080 (127.0.0.1 only)
- Base dir on phone: filesDir/IndigiSnap
- Go library: libindigisnap.so (built by Actions, embedded in APK)
- Debug tag: IndigiSnapDebug (Log.e level)
- Service: com.indigisnap.app.ServerService (foreground, dataSync type)

## Next Actions

1. Wait for green build with debug logging in MainActivity, ServerService, ServerBridge
2. Download APK from Actions artifacts
3. Install on phone via `adb install -r app-debug.apk`
4. Run logcat capture
5. Read output to find where execution stops
