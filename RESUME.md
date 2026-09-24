# How To Resume IndigiSnap

Cold-start orientation. Read this before touching code.

## First 5 Minutes

1. PROJECT.md - what this is
2. TODO.md - where we are
3. DECISIONS_OPEN.md - what's blocked and why
4. ARCHITECTURE.md (incl. Runtime Constraints) - non-obvious facts
5. PRODUCTION_PLAN.md - v1 roadmap
6. PRIORITIZATION_NOTES.md - sequencing rationale
7. CAMERA_SPEC.md - camera capture spec

## Then Check Live State

1. cd /home/deck/IndigiSnap
2. git pull
3. git log --oneline -10
4. Check the GitHub Actions tab - did the last build succeed?
5. Local dev check:
     cd go-backend
     INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server
     Open http://127.0.0.1:8080/browse in browser
6. If APK test needed:
     Download latest artifact from Actions
     adb install -r app-debug.apk
     adb logcat -c
     adb logcat -v threadtime | grep IndigiSnapDebug

## Then Pick Work

- Active work: next item in TODO.md
- Blocked work: DECISIONS_OPEN.md - do not start these
- Batch plan and sequencing: PRIORITIZATION_NOTES.md
- v1 scope: PRODUCTION_PLAN.md

If unsure what to work on, ask: "what's the next action?"
Do not invent work. Do not resolve open decisions silently.

## Working Rules

- File -> Find -> Replace format for all edits.
- One change at a time. Verify after each.
- Terminal via Python heredocs (python3 << 'PYEOF'). Never cat << EOF.
- Go/HTML changes: test locally in browser before pushing.
- Kotlin changes: commit locally, push when wifi available, then build and install.
- Do not assume. Audit before editing.
- Honest about scope. Some features (custom camera) are multi-week efforts.
- Do not defer cheap useful features to v2. Batch them into v1.
- Do not resolve open decisions silently. Ask.

## The Workflow In One Line

Propose a sub-plan, get user approval, then edit with File/Find/Replace
via Python heredocs, verify each step, commit locally, push when possible,
build in CI, install via ADB, test on phone.

## Red Flags To Watch For

- A change "worked" but was not tested on the phone
- An open decision was resolved silently in a code comment
- Something was patched around instead of fixed at the root
- The user says "that's not what I asked for" - stop, re-read the request
- Multiple edits bundled in one step without verification between them

## Common Commands

Run Go server:
  cd /home/deck/IndigiSnap/go-backend
  INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server

Verify Go compiles:
  cd /home/deck/IndigiSnap/go-backend && go vet ./... && go build ./cmd/server

Git status and sync:
  cd /home/deck/IndigiSnap
  git status
  git pull
  git add -A && git commit -m "message" && git push

Install APK:
  adb install -r /path/to/app-debug.apk

Watch phone logs:
  adb logcat -c
  adb logcat -v threadtime | grep IndigiSnapDebug

## Repo Location

Steam Deck: /home/deck/IndigiSnap
GitHub: github.com/calvenwilliams1-pixel/IndigiSnap

## If Something Is Broken And Unclear

1. Do not guess. Stop.
2. State the symptom, what was expected, what happened.
3. Ask the user.
4. Then revise the plan.
5. Only then edit.
