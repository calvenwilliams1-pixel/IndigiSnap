# IndigiSnap — TODO

Live status. Update after every meaningful commit.

Legend: [x] done · [~] in progress · [ ] not started · [!] blocked

---

## Immediate Queue (In Order)

### Pass A — Rotate File Verification [ ]
- Rewrite CameraController.rotateShot:
  - Write to .rotatetmp alongside original
  - Verify: exists, size > 0, inJustDecodeBounds decode check
  - Reset EXIF orientation to NORMAL
  - Atomic rename with explicit boolean check
  - On failure: delete temp, keep original, onShotFailed
- Script at /tmp/pass_a_rotate.py (ready)

### Pass B — Sampled-Decode Thumbnails [ ]
- Split loadOrientedBitmap into loadThumbnail (sampled) and
  loadFullResolution
- generateThumbnail uses loadThumbnail
- Fixes 480 MB heap blowup on sessions with 5+ shots

### Pass C — Unify Delete Model + Multi-Select Commit [ ]
- Long-press multi-select: change delete to mark (like individual)
- New toolbar button: Commit Selected (keeps selected, discards rest)
- Move commitSession to CameraController with keepFilter param
- Grep call sites of deleteShots before changing contract
- Refactor to one underlying commit function

### Pass Q — Capture Presets [ ]
- Renamed from "Quality + Resolution Picker"
- Four presets: Highest Quality, Balanced (default), Storage Saver,
  Fast Capture
- Picker in overflow menu (not top bar)
- Bottom sheet with name + description (no numeric details)
- Persist in SharedPreferences as indigisnap_capture_preset
- Applies to both cameras
- Storage Saver and Fast Capture use both resolution and quality
  reduction
- FIRST: CameraX capability diagnostic to see what's available
- Post-processing as fallback only if native quality control is absent
- Both front and back cameras

### Pass D — CameraTheme.kt + Vector Drawables [ ]
- New file with color constants, button builders, glow helpers,
  thumbnail frames
- 8 vector drawables (close, check, delete, rotate L/R, flash off/on/
  auto, camera switch)
- Neutral fill, color via setColorFilter at runtime
- Layered drawable for glow (not setElevation)

### Pass E — ShutterState Enum [ ]
- Replace onShutterStateChanged(enabled: Boolean) with
  onShutterStateChanged(state: ShutterState)
- Enum: READY, COOLDOWN, DISABLED
- Separate onCaptureStarted() one-shot event for press animation
- Color alone communicates state (no pulse, no continuous animation)

### Pass F — CameraActivity Styling + Gestures [ ]
- Replace makeOverlayButton with CameraTheme helpers
- Shutter three-state color model
- Counter below top bar
- Thumbnail strip uses makeThumbFrame
- Swipe up to toggle camera + feedback pill
- Tap-to-focus + focus indicator (green on success, red on failure)

### Browse-Viewer Pass [ ]
- True fullscreen fix (diagnostic first: getBoundingClientRect vs
  viewport)
- Remove ?t=Date.now() from updateViewerSource (primary lag fix)
- Preload adjacent images
- Server-side rotation cache (Stage B, only if lag remains)
- JPEG quality 100 → 90
- Remove per-tile circles
- Viewer overflow menu (⋮): Favorite, Share, Delete, Info
- Select-mode overflow menu (⋮): Favorite, Share, Delete, Export
  (Rename/Move deferred)
- Native share bridge in MainActivity: shareFile, shareFiles
- FileProvider extension for IndigiSnap/ media root
- Long-press tile → enter select mode with that tile selected
- Long-press viewer image → close viewer, enter grid select mode
- Batch favorite: all-favorited → unfavorite all; else favorite all

### Storage Architecture [ ]
- SAF folder picker on first launch
- Persist URI + resolved path in SharedPreferences
- Mandatory picker on first launch
- Migrate existing app-private files to picked folder
- Open-in-file-manager button in controls bar
- ServerService reads base dir from Intent extra
- Settings option to change storage later (future)

### Orientation Fix [ ]
- Camera preview landscape despite manifest portrait lock
- Add logging: resources.configuration.orientation,
  previewView.display.rotation, imageCapture.targetRotation
- Diagnose root cause (PreviewView surface vs display rotation)
- Fix

### Camera Review Thumbnails Orientation [ ]
- androidx.exifinterface dependency added
- loadOrientedBitmap helper added
- generateThumbnail and updateReviewDisplay updated
- VERIFY in field test

---

## Deferred (Do Not Build Without Explicit Request)

- Play Store distribution prep
- pHash / near-duplicate clustering
- Rating/flag system
- Config export/import
- Side-by-side compare in viewer
- Custom app icon art
- Splash screen polish

---

## Fixed (For Reference)

- [x] Camera FAB calls JS bridge (was calling old input capture path)
- [x] Video upload /data/local/tmp permission error
- [x] CameraX JNI bridge signature mismatch
- [x] Rotate verification EXIF double-rotation
- [x] Portrait shot rotation (targetRotation handling)
- [x] Multi-select batch delete batch file removal
- [x] Review overlay back button
- [x] Pinch-to-zoom in review overlay
- [x] Zoom reset between review sessions

---

## Repo

github.com/calvenwilliams1-pixel/IndigiSnap
Target device: Samsung S25 Ultra, Android 15, SDK 35
