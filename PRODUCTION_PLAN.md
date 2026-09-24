# IndigiSnap - v1 Scope Plan

Authoritative roadmap for reaching a complete, working personal-use app.

## What v1 Is

v1 is the app you use yourself. Single-user, offline, no distribution.
The whole plan is scoped to "does this work well when I open it on my
phone and use it."

v1 is feature-complete when every item in Batch 1, 2, 3, 4, 6.1, 6.2,
6.3 works, verified on the phone, with no crashes on the paths you
actually use.

v1 is NOT: an app store submission, a beta test program, a multi-user
service, or a sharable product. Those live in Section 4 (Only If
Distributing) and stay dormant until explicitly activated.

## Decision Summary

### Resolved
- Camera approach: Option B (custom CameraX). See CAMERA_SPEC.md.
- Personal use only. Distribution deferred.
- Recents: persistent across app restarts.
- Upload button: controls bar next to Sort/Filter.
- File naming: {immediate_folder}_{simplified_date}.{ext}
- Logo priority: prefer logo.<ext>, else alphabetical first.
- Multi-snap discard: inbox folder on disk, not in-memory JS array.

### Open (see DECISIONS_OPEN.md)
- File naming date format (3 options)
- Share-target semantics (copy-only vs. delete-original)
- Multi-snap review screen architecture (JS toggle vs. separate Activity)

## Batch Plan Overview

### Batch 1 - Backend Features (offline-testable in browser)
1.1 Real EXIF reading
1.2 EXIF orientation
1.3 Logo route
1.4 Recent folders tracking

### Batch 2 - UI Features (offline-testable in browser)
2.1 Logo in header
2.2 Recents pills
2.3 Folder browser modal
2.4 Discreet upload button
2.5 Gallery viewer enhancements (zoom, buttons, counter, info)
2.6 Folder-level rotate
2.7 Set logo UI

### Batch 3 - Kotlin Changes (need build to test)
3.1 Splash screen
3.2 Upload move semantics
3.3 Camera capture wiring (CameraX)
3.4 Share target integration
3.5 Splash reads logo (deferred)

### Batch 4 - Camera Implementation (CameraX)
Spec: CAMERA_SPEC.md
Effort: 3-5 weeks part-time
Sequencing: see CAMERA_SPEC.md

### Batch 6 - Trivial Wins + Polish
6.1 Session memory + notes + debug overlay
6.2 EXIF rides-on (formatted strings, EXIF-aware batch rename)
6.3 Search (folder-scoped + global)

## Sequencing (Critical Path)

1. Batch 1 (backend) - offline-friendly, test in browser
2. Batch 2 (UI) - offline-friendly, test in browser
3. Resolve file naming format (Decision 3)
4. Batch 4 (camera, CameraX) - depends on spec and decision
5. Batch 3.3 (camera wiring) - depends on Batch 4
6. Batch 3.1, 3.2, 3.4 - any time
7. Batch 6.1, 6.2, 6.3 - after core features
8. Manual test pass - on the phone, every feature

Parallel-friendly: Batch 6.1 can be done any time, no dependencies.

Estimated total (CameraX path): 8-12 weeks part-time.

## Section 4 - Only If Distributing (Dormant)

This section does nothing until user explicitly says "I'm distributing
this to others." Do not build any of it as part of v1.

If activated, would include:
- Signed release build + keystore management
- Tester distribution (Firebase App Distribution or Play Internal Testing)
- Play Store requirements (privacy policy, data safety form, photo
  permissions declaration, store assets)
- Shared storage migration to /sdcard/Pictures/IndigiSnap/
- Accessibility pass (TalkBack, high contrast, font scaling)
- Tester documentation (TESTER_GUIDE.md, CHANGELOG.md)

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| CameraX work exceeds estimate | High | Decide A vs B honestly. B is locked. |
| Multi-snap loses captures on crash | High | Inbox on capture (CAMERA_SPEC 1) + crash recovery (item 5) |
| Orientation mixing breaks session | Medium | Session persists across orientation (item 2) + folder rotate (item 7) |
| Night mode silently omitted | Medium | Show disabled with reason if hardware lacks support (item 6) |
| Single WebView state leaks | Medium | Explicit reset on discard. Inbox on disk, not JS. |
| Go cross-compile breaks after update | Medium | Pin Go version in CI |
| Rotate writes corrupt file | Medium | Write to temp, verify, atomic rename |

## Definition Of Done (v1)

- [ ] Batch 1 (all four items) working, verified in browser
- [ ] Batch 2 (all seven items, including rotate + set logo) working, verified in browser
- [ ] CAMERA_SPEC.md committed
- [ ] Batch 3.1 (splash) working on phone
- [ ] Batch 3.2 (upload move) working on phone
- [ ] Batch 3.4 (share target) working on phone
- [ ] Batch 4 (CameraX) working per spec, verified on phone
- [ ] Batch 3.3 (camera wiring) working
- [ ] Batch 6.1, 6.2, 6.3 working
- [ ] Inbox pattern implemented, crash-recovery tested
- [ ] All Section 3 decisions resolved
- [ ] Manual test walkthrough passed
- [ ] No known crashes
- [ ] Cold start under 3 seconds

v1 is NOT done when Section 4 exists. Separate phase, separate decision.

## Working Rules

- One change at a time. Verify after each.
- File -> Find -> Replace format for all edits.
- Terminal-only via Python heredocs. Never cat << EOF.
- Test Go/HTML changes in browser before pushing.
- Batch Kotlin changes; commit locally; push when wifi available.
- Do not resolve open decisions silently. Ask.
- Cheap useful features stay in v1. Do not defer for effort-avoidance.
