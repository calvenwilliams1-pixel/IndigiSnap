#!/bin/bash
# verify.sh - check IndigiSnap repo state
# Output: GOOD if all checks pass, FAILED with details otherwise.
# Writes result to clipboard.

set -u
cd "$(dirname "$0")"

FAILED=0
ISSUES=""
WARNINGS=""

# Helper: log issue
issue() {
  FAILED=1
  ISSUES="${ISSUES}FAIL: $1
"
}

warn() {
  WARNINGS="${WARNINGS}WARN: $1
"
}

# --- Check 1: Working directory ---
if [ ! -f "PROJECT.md" ] || [ ! -f "TODO.md" ]; then
  issue "Not in repo root (PROJECT.md / TODO.md missing)"
fi

# --- Check 2: Git clean or expected dirty ---
if [ -d ".git" ]; then
  DIRTY=$(git status --porcelain | wc -l)
  if [ "$DIRTY" -gt 0 ]; then
    warn "Working tree has $DIRTY uncommitted change(s)"
  fi
  BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)
  if [ "$BRANCH" != "main" ]; then
    warn "On branch $BRANCH, expected main"
  fi
fi

# --- Check 3: Go backend builds ---
if [ -d "go-backend" ]; then
  cd go-backend
  if ! go vet ./... > /tmp/verify-go-vet.log 2>&1; then
    issue "go vet failed (see /tmp/verify-go-vet.log)"
  fi
  if ! go build ./cmd/server > /tmp/verify-go-build.log 2>&1; then
    issue "go build failed (see /tmp/verify-go-build.log)"
  fi
  cd ..
else
  issue "go-backend directory missing"
fi

# --- Check 4: Required docs present ---
for f in PROJECT.md TODO.md ARCHITECTURE.md RESUME.md DECISIONS_OPEN.md PRIORITIZATION_NOTES.md PRODUCTION_PLAN.md CAMERA_SPEC.md; do
  if [ ! -f "$f" ]; then
    issue "Missing doc: $f"
  fi
done

# --- Check 5: Key code files present ---
for f in   go-backend/cmd/server/main.go   go-backend/cmd/server/jni.go   go-backend/internal/handlers/info.go   go-backend/internal/handlers/logo.go   go-backend/internal/handlers/view.go   go-backend/internal/handlers/browse.go   go-backend/internal/handlers/actions.go   go-backend/internal/handlers/batch.go   go-backend/internal/meta/meta.go   go-backend/internal/security/security.go   go-backend/internal/ui/interface.html   app/src/main/java/com/indigisnap/app/MainActivity.kt   app/src/main/AndroidManifest.xml ; do
  if [ ! -f "$f" ]; then
    issue "Missing code file: $f"
  fi
done

# --- Check 6: Key routes registered ---
if [ -f "go-backend/cmd/server/main.go" ]; then
  for route in ""/browse"" ""/view/"" ""/image_info/"" ""/logo/"" ""/folder_browser""; do
    if ! grep -q "mux.Handle.*$route" go-backend/cmd/server/main.go; then
      issue "Route not registered: $route"
    fi
  done
fi

# --- Check 7: Batches committed ---
if [ -d ".git" ]; then
  RECENT=$(git log --oneline -10)
  for marker in "Batch 1.1" "Batch 1.2" "Batch 1.3"; do
    if ! echo "$RECENT" | grep -q "$marker"; then
      warn "Recent commits don't mention: $marker"
    fi
  done
fi

# --- Build result ---
{
  echo "==================================================="
  echo "IndigiSnap Verification - $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "Directory: $(pwd)"
  echo "==================================================="
  echo ""
  if [ "$FAILED" -eq 0 ]; then
    echo "GOOD"
    echo ""
    echo "All checks passed."
    if [ -n "$WARNINGS" ]; then
      echo ""
      echo "Warnings (non-blocking):"
      printf "%b" "$WARNINGS"
    fi
  else
    echo "FAILED"
    echo ""
    echo "Issues found:"
    printf "%b" "$ISSUES"
    if [ -n "$WARNINGS" ]; then
      echo ""
      echo "Additional warnings:"
      printf "%b" "$WARNINGS"
    fi
  fi
  echo ""
  echo "==================================================="
} | tee /tmp/verify-output.txt | wl-copy

cat /tmp/verify-output.txt
