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

### Pass Q — Capture Presets [~] built, awaiting field test
- Four presets: Highest Quality, Balanced (default), Storage Saver,
  Fast Capture
- Picker in overflow menu between switch and done
- Dialog with single-choice list; no numeric details exposed
- Persist in SharedPreferences (file=indigisnap_camera, key=capture_preset)
- CameraX diagnostic confirmed: setJpegQuality + setCaptureMode both native
- Implementation:
  - CapturePreset.kt enum (4 entries)
  - CameraController.buildImageCapture() driven by current preset
  - CameraController.setPreset() rebinds ImageCapture on change
  - CameraActivity.showPresetPicker() dialog + SharedPreferences
- Committed: 0016ec4, c161360
- TO VERIFY on phone:
  - Overflow button visible, no collision with X
  - Picker opens with 4 options, current preset checked
  - Selecting preset closes dialog and rebinds (~100-200ms blink)
  - Reopening picker shows new selection
  - Closing and reopening camera persists selection
  - Capture still works in each preset
  - Flash mode unaffected by preset change

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

### Pass VB — Video Thumbnail Generation [ ]
- Status: cleanup + display logic exist, generation is missing entirely
- Files: go-backend/internal/video/video.go, handlers/actions.go
- The .thumbs/<basename>.jpg convention is referenced in video.go,
  browse.go, and actions.go, but nothing ever WRITES a thumbnail.
- Implement VB.1: after video committed from inbox to target folder,
  generate JPEG at {folder}/.thumbs/{basename}.jpg (10 percent frame,
  low quality per CAMERA_SPEC Item 9).
- Decide ffmpeg approach: bundled binary vs on-device shell vs
  skip if unavailable. Android device may not have ffmpeg installed.
- Also implement VB.5: regen on demand when /browse finds a video
  without a thumb.
- Field symptom: video tiles show placeholder emoji, no image.

### Pass VV — Video Viewer White Screen [ ]
- Status: videos don't play in the gallery viewer
- Files: go-backend/internal/handlers/view.go, internal/ui/interface.html
- view.go:137 hardcodes Content-Type "image/jpeg" for ALL files,
  including videos. Browser receives MP4 bytes labeled as JPEG and
  fails to render.
- Fix: detect file extension, set correct Content-Type
  (video/mp4, video/quicktime, etc.) for video files.
- interface.html: viewer has no <video> element handling. Videos
  open in <img>, which cannot play. Add a <video controls> render
  path when .IsVideo is true.
- Field symptom: tapping a video shows white screen or placeholder
  with "gallery image" text.

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
