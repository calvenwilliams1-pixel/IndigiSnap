# Prioritization Notes

Sequencing rationale and batch plan. Numbering matches existing docs; do not introduce a new scheme.

## Batch 1 - Backend Features (offline-testable in browser)

### 1.1 - Real EXIF Reading
- File: go-backend/internal/handlers/info.go
- Dep: github.com/rwcarlsen/goexif/exif
- Parse: Make, Model, ISO, ExposureTime, FNumber, DateTimeOriginal, dimensions
- Return populated JSON. Empty fields when EXIF missing (no crash).
- Date format: keep raw EXIF format ("2006:01:02 15:04:05"). No ISO8601 conversion in v1.
- File close: log the close error, do not swallow silently.
- Videos: skip EXIF, return basic file stats.
- Verification: curl /image_info/<test.jpg> returns real values. JPEG without EXIF returns empty fields. go vet passes.

### 1.2 - EXIF Orientation
- File: go-backend/internal/handlers/view.go
- Dep: github.com/disintegration/imaging
- Read orientation tag (1/3/6/8). Rotate on serve.
  - 3 -> Rotate180, 6 -> Rotate270, 8 -> Rotate90, others -> serve as-is
- Re-encode at jpeg.Options{Quality: 100}. Note: not lossless. Comment in code that v2 could use quality 90 or read original.
- Metadata loss: rotation loses EXIF/ICC on the SERVED copy. Original on disk unchanged. Comment for future.
- Non-JPEG: serve as-is.
- Verification: portrait JPEG displays correct orientation in browser.

### 1.3 - Logo Route
- New file: go-backend/internal/handlers/logo.go
- meta.go: GetLogoPath, GetLogoURL
- Priority: prefer file named logo.<ext>, else first alphabetically, else "".
- Route: GET /logo/<path>. Validate with IsSafePath, ensure prefix is BASE_DIR/logo/.
- browse.go: populate TemplateData.LogoURL
- SVG allowed for personal use; comment in code for multi-user sanitization.
- Verification: logo.png shows in header. Only banana.png shows banana. Both logo.png and apple.png show logo. No folder -> placeholder. Path traversal blocked.

### 1.4 - Recent Folders Tracking
- File: go-backend/internal/meta/meta.go
- Storage: BASE_DIR/.indigisnap_recents.json. Raw paths (no URL encoding in storage).
- Add on /browse visit for real folders (not root, not favorites).
- Dedupe by path. Prepend newest. Trim to 5.
- Stale entry validation: os.Stat. os.IsNotExist drops; other errors keep (log at debug).
- Template: use {{ .Path | urlquery }}. If nested paths break, fall back to per-segment encoding template func.
- Mutex around load+save to avoid race.
- Verification: nested path pill navigates. Restart persists. Deleted folder drops. Space in name works. Hover shows full path.

## Batch 2 - UI Features (offline-testable in browser)

### 2.1 - Logo In Header
- Verify Batch 1.3 populates LogoURL correctly.

### 2.2 - Recents Pills
- Verify Batch 1.4 populates Recents correctly.

### 2.3 - Folder Browser Modal
- File: interface.html
- Modal: folder list, Up, Select, Cancel
- JS: loadFolderBrowser(path) -> /folder_browser?path=
- Replace prompt() in batchMove()
- Verification: select files -> Move -> modal -> navigate -> select

### 2.4 - Discreet Upload Button
- File: interface.html
- Small icon in controls bar next to Sort/Filter. Not a FAB.
- Opens <input type="file" multiple>. No capture attribute.
- Submits to /upload with current folder context.
- Verification: button visible, picker opens, files upload

### 2.5 - Gallery Viewer Enhancements
- File: interface.html
- Pinch-to-zoom, double-tap-zoom
- Delete / Favorite / Share buttons in viewer
- Photo counter ("3 of 47")
- Info button -> /image_info/<path> -> formatted panel (human-readable strings)
- Verification: each individually

### 2.6 - Folder-Level Rotate
- Backend route: POST /rotate_picture/<path>
- File: go-backend/internal/handlers/actions.go (add handler)
- Rotate 90/180/270, rewrite file on disk, use disintegration/imaging
- UI: rotate button in image viewer or folder tile menu
- Scope: one route, one button. No batch-rotate.
- Verification: rotate a photo, verify it persists after refresh

### 2.7 - Set Logo UI
- New route: POST /set_logo. Accepts multipart file. Saves to BASE_DIR/logo/logo.<ext>.
- UI: "Set Logo" option in folder-level menu (root folder or global)
- Opens file picker, submits to /set_logo, redirects back
- Note: app-private storage means user cannot drop logo via file manager. This is the in-app path.
- Verification: pick image, header refreshes with new logo. Persist after restart.

## Batch 3 - Kotlin Changes (need build to test)

### 3.1 - Splash Screen
- androidx.core:core-splashscreen
- Branded: ghost emoji + INDIGISNAP neon
- Fade transition to WebView

### 3.2 - Upload Move Semantics
- WebView file chooser returns URI -> copy to app-private temp -> POST -> delete original via ContentResolver
- Requires runtime permission for source location

### 3.3 - Camera Capture Wiring (CameraX)
- Depends on Batch 4 spec
- Kotlin CameraX integration into custom capture Activity or WebView-embedded surface
- Writes per-shot to .inbox/<session-id>/ (see CAMERA_SPEC.md item 1)
- Communicates session-id to JS for review UI

### 3.4 - Share Target Integration
- Manifest intent filter for ACTION_SEND / ACTION_SEND_MULTIPLE
- Kotlin handler: extract URIs, copy to current folder, optionally delete originals
- Depends on Share-Target Semantics decision (see DECISIONS_OPEN.md)
- Verification: share from Gallery, verify file moves to IndigiSnap

### 3.5 - Splash Reads Logo (Deferred / Optional)
- MainActivity splash HTML updated to fetch /logo after server up
- Deferred until after v1 ships. Splash is ~200ms; header already shows logo.

## Batch 4 - Camera (CameraX)

Spec: CAMERA_SPEC.md
Effort: 3-5 weeks part-time
Sequencing per CAMERA_SPEC.md.

Do not start until:
- File naming format decided
- Multi-snap review architecture decided
- CAMERA_SPEC.md committed

## Batch 6 - Trivial Wins + Polish

### 6.1 - Session Memory + Notes + Debug Overlay
- Session memory: localStorage for last folder, view mode, grid size
- Per-folder notes: extend .indigisnap_meta.json with notes field
  - POST /set_notes/<path>
  - Visible at top of folder when present
  - Edit via folder menu
- Debug overlay L1: corner badge (green/red server status), polls /health every 10s
- Debug overlay L2: long-press badge -> full-screen log viewer, copy-to-clipboard

### 6.2 - EXIF Rides-On
- Human-readable EXIF strings: "1/125s . f/2.8 . ISO 400 . 24mm"
- Info panel uses formatted strings, not raw tags
- EXIF-aware batch rename: pattern tokens {n} {date} {time} {folder} {camera} {iso}
  - Server-side substitution
  - Preview before commit

### 6.3 - Search
- Folder-scoped substring search: input in controls bar
  - Filters current folder client-side
  - Matched substring bolded
- Global search toggle: opt-in, server walks tree, returns JSON
  - New route: GET /search?q=<query>
  - Distinct results section

### 6.4 - Multi-Snap Inbox Architecture
- Now folded into Batch 4 spec items 1 and 5
- Not a separate batch item

## Lessons From Prioritization Review

- Do not build features that depend on unresolved decisions (camera presets before Batch 4, EXIF-write before deciding it's wanted).
- Perceptual hashing is a different algorithm, not an extension of MD5 - its own batch if wanted.
- Skip side-by-side compare in viewer - bad phone ergonomics, not a real gap.
- Single WebView means every temp collection needs explicit reset on discard.
- EXIF read (goexif) and EXIF write (separate library) are different scopes.
- Cheap useful features stay in v1. Do not defer for effort-avoidance.

## Deferred (Do Not Build in v1)

See DECISIONS_OPEN.md "Deferred" section:
- Shared storage migration
- EXIF write support
- pHash / near-duplicate clustering
- Rating/flag system
- Config export/import
