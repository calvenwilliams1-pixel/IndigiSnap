# Architecture

## Root
- PROJECT.md — AI context prompt
- TODO.md — status tracker
- ARCHITECTURE.md — this file

## app/ (Android shell)
- MainActivity.kt — launches WebView
- ServerBridge.kt — JNI loader for Go .so
- ServerService.kt — foreground service
- AndroidManifest.xml — permissions + service
- jniLibs/*/libindigisnap.so — compiled Go

## go-backend/ (HTTP server)
- cmd/server/main.go — entry point, routes
- cmd/server/jni.go — JNI export functions
- internal/security/security.go — path safety
- internal/meta/meta.go — metadata, favorites
- internal/ui/ui.go — template renderer
- internal/ui/interface.html — the app UI
- internal/video/video.go — ffmpeg wrappers
- internal/handlers/browse.go — /browse route
- internal/handlers/view.go — /view route
- internal/handlers/actions.go — POST actions
