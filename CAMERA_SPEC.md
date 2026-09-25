# CAMERA_SPEC.md

Authoritative spec for the IndigiSnap camera capture flow.

Approach: Option B (custom CameraX). Option A (system camera) cannot
produce the interaction described below - it's a capability gap, not
a cost tradeoff.

## The Interaction (Why B)

1. Open camera from inside a target folder (e.g. Japan/Tokyo/Shrine)
2. Preview stays open, user takes multiple shots rapidly - no closing
   or reopening the camera between shots
3. Shoot while walking, without fiddling with the phone between shots
4. When ready, review the batch
5. SELECT THE ONES TO DELETE (deletion is the selection action, not saving)
6. Confirm
7. Everything remaining is saved to the same folder the camera was launched from

## Item 1 - Capture-Time Storage

Each shutter press writes the file IMMEDIATELY to disk.

- Path: filesDir/IndigiSnap/.inbox/<session-id>/
- Written by Kotlin directly to the filesystem
- NOT posted to the Go HTTP server per shot
- NOT held only in an in-memory JS or Kotlin array

Why: Per-shot latency must stay low enough for rapid-fire shooting
while walking. A network round-trip per photo is not acceptable.

Implication: The .inbox/ folder is the source of truth for pending
captures. JS holds only filenames and session-id, never file data.

## Item 2 - Session Persistence Across Orientation Changes

A single capture session can include a mix of portrait and landscape shots.

- Session must not reset, close, or split when device orientation
  changes mid-session
- Each photo stores its own orientation/EXIF rotation independently
- Review grid displays each thumbnail in its own correct orientation,
  not a forced uniform grid orientation

Why: User shoots while walking. Some shots will be portrait, some
landscape. The session is a continuous experience, not a per-orientation
experience.

## Item 3 - Review UI Is Select-To-Delete

After ending the capture session:

- Show a grid of every shot taken this session
- User taps/highlights the ones they want REMOVED
- On confirm, those are deleted from the inbox folder
- Everything NOT marked is what gets kept

This is inverted from typical "select to save" UIs. Do not implement
as "check the ones to save." The default state is keep-everything;
the action is marking exceptions for deletion.

Why: User's mental model is "I took a bunch of shots, I want to
remove the bad ones." The common case is "keep almost all of them."

## Item 4 - Commit Behavior

On confirm:

- Every remaining (non-deleted) file in the inbox session folder
  MOVES into the folder the camera was launched from
- Not root. Not a prompt. The folder the user was already in when
  they tapped the camera button.
- The now-empty inbox session folder is deleted

Why: The user launched the camera from a specific folder context.
That context is the destination.

## Item 5 - Discard / Crash Recovery

If the app is killed or backgrounded without a commit:

- The inbox session folder persists on disk (writes happen on
  capture, not commit)
- On next app start, detect orphaned inbox sessions
- Prompt to resume review or discard

Why: Writes on capture means a crash doesn't lose photos. But it
does leave orphaned data. User needs a recovery path; app needs a
cleanup path.

## Item 6 - Full Native Camera Feature Parity

Not simplified. A prior build attempt stripped features; that must
not happen again.

Standard CameraX (no gap expected):
- Zoom (pinch and/or slider)
- Front/back camera switch
- Flash toggle (on/off/auto)

Night mode - NOT guaranteed on all hardware:
- CameraX night capture depends on the device supporting the
  CameraX Extensions API (vendor-provided; varies by chipset/OEM)
- At app start, query ExtensionsManager for night-mode support on
  the actual target device (Samsung, Android 14)
- If supported: expose the toggle normally
- If NOT supported: show the toggle VISIBLY BUT DISABLED, with a
  stated reason ("not supported on this device")
- Never silently omit it. User must be able to tell the difference
  between "not built" and "not supported by hardware"

## Item 7 - Folder-Level Rotate (Separate From Camera)

Independent of the capture flow. A rotate action on saved photos
within a folder.

- Rotate 90 / 180 / 270 degrees
- Applied via the Go backend
- New route: POST /rotate_picture/<path>
- Uses disintegration/imaging (same library as EXIF-orientation serving)
- Rotates the file on disk, so the corrected orientation persists
- Available from the folder view (and/or the image viewer)

Why: Since portrait/landscape can mix within a single session
(item 2), some photos may display in the wrong orientation later.
This is the folder-side correction path.

Scope discipline: Backend route + one button in the viewer.
Do not overcomplicate. No batch-rotate for v1.

## Sequencing Within Batch 4

1. CameraX dependencies + preview surface
2. Shutter + rapid-fire writes to inbox (item 1)
3. Session persistence across orientation (item 2)
4. Review grid with select-to-delete (item 3)
5. Commit = move + delete inbox (item 4)
6. Crash recovery prompt (item 5)
7. Feature parity: zoom, camera switch, flash (item 6)
8. Night mode with hardware-aware disabled state (item 6)
9. Test matrix on Samsung Android 14

## Do Not Start Batch 4 Code Until

- File naming format decided (see DECISIONS_OPEN.md)
- Multi-snap review screen architecture decided (see DECISIONS_OPEN.md)
- This spec committed


## Item 8 - File Naming Format

Camera captures use the format:

    {immediate_folder}_{MM-DD-YY}.{ext}

Examples (in folder Family/Reunion, photos taken 2026-09-24):
- Family_Reunion_09-24-26.jpg
- Family_Reunion_09-24-26_1.jpg   (second photo same day)
- Family_Reunion_09-24-26_2.jpg   (third photo same day)

The immediate folder name is the leaf folder where the camera was launched.
The date is the capture date in MM-DD-YY format (US-style month-day-year).
Duplicate names on the same day append _1, _2, _3, etc.

The same format applies to video files, using the video extension.

## Item 9 - Video Thumbnail Generation

When a video is captured and committed from the inbox to a folder, a
thumbnail is generated automatically.

Spec:
- Location: hidden .thumbs/ subfolder inside the same folder as the video
- Path: {folder}/.thumbs/{video_basename}.jpg
- Format: JPEG, low quality is fine (target ~160-320px wide, quality 70-80)
- Frame: 10 percent into the video (matches original Flask behavior)
- Generation timing: on commit (when video moves from inbox to final folder)
- Failure handling: if generation fails, video still commits, grid shows
  placeholder emoji until thumbnail is regenerated

Cleanup:
- When a video is deleted (single or batch), its thumbnail is deleted too
- On every /browse call, an orphan sweep runs on .thumbs/ folders,
  deleting thumbnails whose video no longer exists
- The sweep extends the existing CleanOrphanThumbnails logic in the
  video package

Display:
- Video tiles in the grid look for .thumbs/{basename}.jpg
- If present, served via existing /view/ route as <img src>
- If missing, fall back to the placeholder emoji

Regeneration:
- If the thumbnail is missing but the video exists, regenerate on demand
  (runs as a background task when /browse detects the gap)
