# Open Decisions

Do not build against these until resolved. When resolved, move the item to the Resolved section below with a date and rationale.

## Open

### File Naming Date Format
Camera captures name files as: {immediate_folder}_{simplified_date}.{ext}
Example: Shrine_09-24-26_15-14.jpg
Date format options:
- A: MM-DD-YY_HH-MM      -> Shrine_09-24-26_15-14.jpg
- B: MMDDYY_HHMM         -> Shrine_092426_1514.jpg
- C: YYYY-MM-DD_HHMM     -> Shrine_2026-09-24_1514.jpg
Blocks: Batch 4 (CameraX implementation)
Notes: Rapid-fire shooting may need a counter suffix (_01, _02) for same-second collisions.

### Share-Target Semantics
When sharing photos/videos from the phone's gallery to IndigiSnap:
- Option 1: Copy then delete original. True move. Requires MANAGE_MEDIA permission (Android 11+) and a one-time system permission prompt.
- Option 2: Copy only. Original stays in gallery. No permission prompt.
Blocks: Batch 3.4 (Share target integration)
Notes: Option 1 matches the stated ask (remove from camera roll) but has a permission cost. Option 2 is simpler but requires manual cleanup.

### Multi-Snap Review Screen Architecture
The multi-snap review UI can be:
- JS view toggle in the main WebView (consistent with existing modals, but state persists across toggles - needs explicit reset)
- Separate WebView or Activity (state dies with the view, but inconsistent with app architecture)
Blocks: Batch 4 (CameraX implementation)
Notes: Either way, per CAMERA_SPEC.md item 1, files write to inbox on capture, not in-memory. So the state that matters is the file list on disk, not the JS array. JS state is a UI concern only.

## Resolved

### Camera Approach (Resolved: Option B, custom CameraX)
Reason: The described capture flow (open from folder, rapid-fire shots without reopening camera, review batch, select-to-delete, commit rest to same folder) requires a persistent camera session with our own UI. Option A (system camera) reopens per shot and has no review step. This is a capability gap, not a cost tradeoff.
Date: 2026-09-24

### Personal Use vs. Distribution (Resolved: Personal use only)
Reason: Distribution is a future want, not current need. All distribution-related work (signed builds, testers, Play Store, accessibility, shared storage) deferred. Section 4 of PRODUCTION_PLAN.md stays dormant until user explicitly says otherwise.
Date: 2026-09-24

### Recents Persistence (Resolved: Persistent)
Reason: User wants recents to survive app restarts. File-backed at BASE_DIR/.indigisnap_recents.json, last 5 folders, deduped, newest first.
Date: 2026-09-24

### Upload Button Placement (Resolved: Controls bar)
Reason: Next to Sort/Filter. Avoids a third floating button. User can relocate later if uncomfortable.
Date: 2026-09-24

### Multi-Snap Discard Mechanics (Resolved: Inbox folder)
Reason: Files write to filesDir/IndigiSnap/.inbox/<session-id>/ on capture, written by Kotlin directly. JS holds only filenames. Discard = delete inbox session folder. Commit = move files from inbox to current folder. Crash recovery: orphaned inbox folders detected on app start, prompt to resume or discard.
Date: 2026-09-24

### Logo Priority (Resolved: Two-tier)
Reason: Prefer file named logo.<ext>, else first alphabetically. Simpler than multi-name priority list, matches common convention.
Date: 2026-09-24

## Deferred (Do Not Build)

### Shared Storage Migration
Currently everything is app-private (filesDir/IndigiSnap/). Migration to /sdcard/Pictures/IndigiSnap/ (visible to other apps) requires MediaStore integration and permissions. Deferred until distribution or explicit request.

### EXIF Write Support
Reading EXIF (goexif) is different from writing (separate library, corruption risk). Not needed for v1 features. Deferred.

### pHash / Near-Duplicate Clustering
Current /find_duplicates uses exact MD5 match. Perceptual hashing is a different algorithm, its own project. Deferred.

### Rating/Flag System
Favorites already exist as a JSON-backed star system. A second dimension needs its own store and risks duplicating the concept. Hold off until felt need during use.

### Config Export/Import
Storage is on-device, code is on GitHub. Solves migrating to a new phone, a rare event. Deferred.
